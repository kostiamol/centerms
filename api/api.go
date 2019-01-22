package api

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/kostiamol/centerms/svc"

	"github.com/kostiamol/centerms/cfg"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"fmt"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
)

// cfgProvider deals with device configurations.
type cfgProvider interface {
	GetDevCfg(id string) (*svc.DevCfg, error)
	SetDevInitCfg(*svc.DevMeta) (*svc.DevCfg, error)
	SetDevCfg(id string, c *svc.DevCfg) error
	PublishCfgPatch(c *svc.DevCfg, channel string) (int64, error)
}

// dataProvider deals with device data.
type dataProvider interface {
	GetDevsData() ([]svc.DevData, error)
	GetDevData(id string) (*svc.DevData, error)
	SaveDevData(*svc.DevData) error
}

// API is used to deal with user's queries from the web client (dashboard).
type API struct {
	log                 *logrus.Entry
	pubChan             string
	portRPC             int32
	portREST            int32
	cfgProvider         cfgProvider
	dataProvider        dataProvider
	retry               time.Duration
	isProd              bool
	redirectHTTPToHTTPS bool
	allowedHost         string
}

// APICfg is used to initialize an instance of API.
type APICfg struct {
	Log                 *logrus.Entry
	PubChan             string
	PortRPC             int32
	PortREST            int32
	CfgProvider         cfgProvider
	DataProvider        dataProvider
	Retry               time.Duration
	IsProd              bool
	RedirectHTTPToHTTPS bool
	AllowedHost         string
}

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
	}
}

// Run launches the service by running goroutines for listening the service termination and queries from the web client.
func (a *API) Run() {
	a.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("is running on rpc port: [%d], rest port: [%d]", a.portRPC, a.portREST)

	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
		}
	}()

	a.runRPCServer()

	var m *autocert.Manager
	var httpsSrv *http.Server
	if a.isProd {
		hostPolicy := func(ctx context.Context, host string) error {
			allowedHost := a.allowedHost
			if host == allowedHost {
				return nil
			}
			return fmt.Errorf("acme/autocert: only %s host is allowed", allowedHost)
		}

		dataDir := "./crt"
		m = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      autocert.DirCache(dataDir),
		}

		httpsSrv = a.makeHTTPServer()
		httpsSrv.Addr = ":" + fmt.Sprint(a.portREST)
		httpsSrv.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}

		go func() {
			fmt.Printf("starting HTTPS server on %s\n", httpsSrv.Addr)
			a.log.Fatal(httpsSrv.ListenAndServe())
		}()
	}

	var httpSrv *http.Server
	if a.redirectHTTPToHTTPS {
		httpSrv = a.makeHTTPToHTTPSRedirectServer()
	} else {
		httpSrv = a.makeHTTPServer()
	}
	// allow autocert handle Let'a Encrypt callbacks over http
	if m != nil {
		httpSrv.Handler = m.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = ":" + fmt.Sprint(a.portREST)
	if err := httpSrv.ListenAndServe(); err != nil {
		a.log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}

	// allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS", "PATCH"})
	// allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	// http.ListenAndServe(a.addrREST.Host+":"+fmt.Sprint(a.addrREST.PortWS), handlers.CORS(allowedMethods, allowedHeaders)(r))
	// a.log.Fatal(httpSrv.ListenAndServe())
}

func (a *API) initRouter(r *mux.Router) {
	r.Handle("/devices", jwtMiddleware.Handler(adapt(a.getDevsDataHandler, a.recoveryAdapter))).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", adapt(a.getDevDataHandler, a.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/cfg", adapt(a.getDevCfgHandler, a.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/cfg", adapt(a.patchDevCfgHandler, a.recoveryAdapter)).Methods(http.MethodPatch)
	// for Prometheus
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	// for auth + token
	r.Handle("/get-token", getTokenHandler).Methods(http.MethodGet)
}

func (a *API) makeHTTPServer() *http.Server {
	r := mux.NewRouter()
	a.initRouter(r)
	return makeServerFromMux(r)
}

func makeServerFromMux(r *mux.Router) *http.Server {
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}
}

func (a *API) makeHTTPToHTTPSRedirectServer() *http.Server {
	r := mux.NewRouter()
	r.Handle("/", adapt(a.redirectHandler, a.recoveryAdapter))
	return makeServerFromMux(r)
}

// https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
type adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func adapt(hf http.HandlerFunc, adapters ...adapter) http.Handler {
	for _, adapter := range adapters {
		hf = adapter(hf)
	}
	return hf
}

func (a *API) recoveryAdapter(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				a.log.WithFields(logrus.Fields{
					"func":  "recoveryAdapter",
					"event": cfg.EventPanic,
				}).Errorf("%s", r)
			}
		}()
		h.ServeHTTP(wr, r)
	})
}
