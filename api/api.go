package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kostiamol/centerms/cfg"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
)

// DevID is used for device's id.
type DevID string

// DevMeta is used to store device metadata: it's type, name (model) and MAC address.
type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

// DevData is used to store time of the request, device's metadata and the data it transfers.
type DevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// DevCfg holds device's MAC address and config.
type DevCfg struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

// cfgProvider deals with device configurations.
type cfgProvider interface {
	//GetDevCfg(id DevID) (*DevCfg, error)
	SetDevInitCfg(m *DevMeta) (*DevCfg, error)
	//SetDevCfg(id DevID, c *DevCfg) error
	//Publish(msg interface{}, channel string) (int64, error)
}

// dataProvider deals with device data.
type dataProvider interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id DevID) (*DevData, error)
	SaveDevData(d *DevData) error
	GetDevMeta(id DevID) (*DevMeta, error)
	SetDevMeta(m *DevMeta) error
}

// API is used to deal with user's queries from the web client (dashboard).
type API struct {
	host                string
	rpcPort             int
	restPort            int
	cfgProvider         cfgProvider
	dataProvider        dataProvider
	log                 *logrus.Entry
	retry               time.Duration
	pubChan             string
	agent               *consul.Agent
	agentName           string
	ttl                 time.Duration
	isProd              bool
	redirectHTTPToHTTPS bool
	allowedHost         string
}

// NewAPI creates and initializes a new instance of API component.
func NewAPI(host string, rpcPort, restPort int, cfg cfgProvider, log *logrus.Entry, retry time.Duration,
	pubChan, agentName string, ttl time.Duration) *API {

	return &API{
		host:        host,
		rpcPort:     rpcPort,
		restPort:    restPort,
		cfgProvider: cfg,
		log:         log.WithFields(logrus.Fields{"component": "api"}),
		retry:       retry,
		pubChan:     pubChan,
		agentName:   agentName,
		ttl:         ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination and queries from the web client.
func (a *API) Run() {
	a.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("running on host: [%s], rpc port: [%d], rest port: [%d]", a.host, a.rpcPort, a.restPort)

	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
		}
	}()

	a.runConsulAgent()

	a.runRPC()

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
		httpsSrv.Addr = a.host + ":" + fmt.Sprint(a.restPort)
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

	httpSrv.Addr = a.host + ":" + fmt.Sprint(a.restPort)
	if err := httpSrv.ListenAndServe(); err != nil {
		a.log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}

	// allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS", "PATCH"})
	// allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	// http.ListenAndServe(a.addrREST.Host+":"+fmt.Sprint(a.addrREST.Port), handlers.CORS(allowedMethods, allowedHeaders)(r))
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

func (a *API) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	agent := &consul.AgentServiceRegistration{
		Name: a.agentName,
		Port: a.restPort,
		Check: &consul.AgentServiceCheck{
			TTL: a.ttl.String(),
		},
	}
	a.agent = c.Agent()
	if err := a.agent.ServiceRegister(agent); err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	go a.updateTTL(a.check)
}

func (a *API) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (a *API) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(a.ttl / 2)
	for range t.C {
		a.update(check)
	}
}

func (a *API) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		a.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := a.agent.UpdateTTL("svc:"+a.agentName, "", health); err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Error(err)
	}
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