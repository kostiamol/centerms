package api

import (
	"net/http"
	"time"

	"github.com/kostiamol/centerms/svc"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/cfg"
	"github.com/rs/cors"

	"fmt"
)

type (
	cfgProvider interface {
		GetDevCfg(id string) (*svc.DevCfg, error)
		SetDevInitCfg(*svc.DevMeta) (*svc.DevCfg, error)
		SetDevCfg(id string, c *svc.DevCfg) error
		PublishCfgPatch(c *svc.DevCfg, channel string) (int64, error)
	}

	dataProvider interface {
		GetDevsData() ([]svc.DevData, error)
		GetDevData(id string) (*svc.DevData, error)
		SaveDevData(*svc.DevData) error
	}

	// API is used to deal with user's queries from the web client (dashboard).
	API struct {
		appID        string
		log          *logrus.Entry
		pubChan      string
		portRPC      int32
		portREST     int32
		cfgProvider  cfgProvider
		dataProvider dataProvider
		retry        time.Duration
		router       *mux.Router
		token        *tokenValidator
		metric       *metric
		publicKey    string
	}

	// APICfg is used to initialize an instance of API.
	APICfg struct {
		AppID        string
		Log          *logrus.Entry
		PubChan      string
		PortRPC      int32
		PortREST     int32
		CfgProvider  cfgProvider
		DataProvider dataProvider
		Retry        time.Duration
		PublicKey    string
	}
)

// NewAPI creates and initializes a new instance of API.
func NewAPI(c *APICfg) *API {
	return &API{
		log:          c.Log.WithFields(logrus.Fields{"component": "api"}),
		pubChan:      c.PubChan,
		portRPC:      c.PortRPC,
		portREST:     c.PortREST,
		cfgProvider:  c.CfgProvider,
		dataProvider: c.DataProvider,
		retry:        c.Retry,
		publicKey:    c.PublicKey,
	}
}

// Run launches the service by running goroutines for listening the service termination and queries
// from the web client.
func (a *API) Run() {
	a.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("is running on: rpc port [%d], rest port [%d]", a.portRPC, a.portREST)

	var err error
	a.token, err = newTokenValidator(a.publicKey)
	if err != nil {
		a.log.Fatalf("")
	}

	a.router = mux.NewRouter()
	a.metric = newMetric(a.appID)
	a.registerRoutes()
	a.serve()
}

func (a *API) registerRoutes() {
	middleware := []func(next http.HandlerFunc, name string) http.HandlerFunc{
		a.requestLogger,
		a.token.validator,
		a.metric.timeTracker,
	}

	a.registerRoute(http.MethodGet, "/health", a.health)
	a.registerRoute(http.MethodGet, "/metrics", a.metric.httpRouterHandler())

	a.registerRoute(http.MethodGet, "/v1/token", getTokenHandler)

	a.registerRoute(http.MethodGet, "/v1/device", a.getDevsDataHandler, middleware...)
	a.registerRoute(http.MethodGet, "/v1/device/{id}/data", a.getDevDataHandler, middleware...)
	a.registerRoute(http.MethodGet, "/v1/device/{id}/config", a.getDevCfgHandler, middleware...)
	a.registerRoute(http.MethodPatch, "/v1/device/{id}/config", a.patchDevCfgHandler, middleware...)
}

func (a *API) serve() {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "HEAD", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})

	s := &http.Server{
		Handler:      c.Handler(a.router),
		Addr:         ":" + fmt.Sprint(a.portREST),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	if err := s.ListenAndServe(); err != nil {
		a.log.Fatalf("ListenAndServe() failed: ", err)
	}
}
