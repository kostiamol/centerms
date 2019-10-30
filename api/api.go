package api

import (
	"net/http"
	"time"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/svc"

	"fmt"

	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/cfg"
	"github.com/rs/cors"
)

type (
	// CfgProvider is a contract for the configuration provider.
	CfgProvider interface {
		GetDevCfg(id string) (*svc.DevCfg, error)
		SetDevInitCfg(*svc.DevMeta) (*svc.DevCfg, error)
		SetDevCfg(id string, c *svc.DevCfg) error
	}

	// DataProvider is a contract for the data provider.
	DataProvider interface {
		GetDevsData() ([]svc.DevData, error)
		GetDevData(id string) (*svc.DevData, error)
		SaveDevData(*svc.DevData) error
	}

	// Publisher .
	Publisher interface {
		Publish(msg interface{}, channel string) (int64, error)
	}

	// Cfg is used to initialize an instance of api.
	Cfg struct {
		AppID        string
		Log          log.Logger
		PubChan      chan<- *svc.DevCfg
		PortRPC      uint32
		PortREST     uint32
		CfgProvider  CfgProvider
		DataProvider DataProvider
		Retry        time.Duration
		PublicKey    string
		PrivateKey   string
	}

	// api includes both rest and grpc.
	api struct {
		appID        string
		log          log.Logger
		pubChan      chan<- *svc.DevCfg
		portRPC      uint32
		portREST     uint32
		cfgProvider  CfgProvider
		dataProvider DataProvider
		retry        time.Duration
		router       *mux.Router
		token        *tokenValidator
		metric       *metric
		publicKey    string
		privateKey   string
	}
)

// New creates and initializes a new instance of api.
func New(c *Cfg) *api { // nolint
	return &api{
		log:          c.Log.With("component", "api"),
		pubChan:      c.PubChan,
		portRPC:      c.PortRPC,
		portREST:     c.PortREST,
		cfgProvider:  c.CfgProvider,
		dataProvider: c.DataProvider,
		retry:        c.Retry,
		publicKey:    c.PublicKey,
		privateKey:   c.PrivateKey,
	}
}

// Run launches the service by running goroutines for listening to the service termination and queries
// from the web client.
func (a *api) Run() {
	a.log.With("event", cfg.EventComponentStarted).
		Infof("is running on rpc port [%d] rest port [%d]", a.portRPC, a.portREST)

	var err error
	if a.token, err = newTokenValidator(a.publicKey); err != nil {
		a.log.Fatalf("newTokenValidator(): %s", err)
	}

	a.metric = newMetric(a.appID)

	go a.serveRPC()

	a.router = mux.NewRouter()
	a.registerRoutes()
	a.serveHTTP()
}

func (a *api) registerRoutes() {
	middleware := []func(next http.HandlerFunc, name string, l log.Logger) http.HandlerFunc{
		requestLogger,
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

func (a *api) serveHTTP() {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "HEAD", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})

	s := &http.Server{
		Handler: c.Handler(a.router),
		Addr:    fmt.Sprintf(":%d", a.portREST),
	}

	if err := s.ListenAndServe(); err != nil {
		a.log.Fatalf("ListenAndServe(): %s", err)
	}
}
