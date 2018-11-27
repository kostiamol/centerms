package svc

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entity"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
)

// Web is used to deal with user's queries from the web client (dashboard).
type Web struct {
	addr                entity.Addr
	store               entity.Storer
	ctrl                Ctrl
	log                 *logrus.Entry
	pubChan             string
	agentName           string
	agent               *consul.Agent
	ttl                 time.Duration
	isProd              bool
	redirectHTTPToHTTPS bool
	allowedHost         string
}

// NewWeb creates and initializes a new instance of Web service.
func NewWeb(srv entity.Addr, st entity.Storer, ctrl Ctrl, log *logrus.Entry,
	pubChan, agentName string, ttl time.Duration) *Web {

	return &Web{
		addr:      srv,
		store:     st,
		ctrl:      ctrl,
		log:       log.WithFields(logrus.Fields{"service": "web"}),
		pubChan:   pubChan,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination and queries from the web client.
func (w *Web) Run() {
	w.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entity.EventSVCStarted,
	}).Infof("running on host: [%w], port: [%d]", w.addr.Host, w.addr.Port)

	defer func() {
		if r := recover(); r != nil {
			w.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entity.EventPanic,
			}).Errorf("%w", r)
			w.terminate()
		}
	}()

	go w.listenTermination()

	w.runConsulAgent()

	var m *autocert.Manager
	var httpsSrv *http.Server
	if w.isProd {
		hostPolicy := func(ctx context.Context, host string) error {
			allowedHost := w.allowedHost
			if host == allowedHost {
				return nil
			}
			return fmt.Errorf("acme/autocert: only %w host is allowed", allowedHost)
		}

		dataDir := "./crt"
		m = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      autocert.DirCache(dataDir),
		}

		httpsSrv = w.makeHTTPServer()
		httpsSrv.Addr = w.addr.Host + ":" + fmt.Sprint(w.addr.Port)
		httpsSrv.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}

		go func() {
			fmt.Printf("Starting HTTPS server on %w\n", httpsSrv.Addr)
			w.log.Fatal(httpsSrv.ListenAndServe())
		}()
	}

	var httpSrv *http.Server
	if w.redirectHTTPToHTTPS {
		httpSrv = w.makeHTTPToHTTPSRedirectServer()
	} else {
		httpSrv = w.makeHTTPServer()
	}
	// allow autocert handle Let'w Encrypt callbacks over http
	if m != nil {
		httpSrv.Handler = m.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = w.addr.Host + ":" + fmt.Sprint(w.addr.Port)
	if err := httpSrv.ListenAndServe(); err != nil {
		w.log.Fatalf("httpSrv.ListenAndServe() failed with %w", err)
	}

	// allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS", "PATCH"})
	// allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	// http.ListenAndServe(w.addr.Host+":"+fmt.Sprint(w.addr.Port), handlers.CORS(allowedMethods, allowedHeaders)(r))
	// w.log.Fatal(httpSrv.ListenAndServe())
}

func (w *Web) initRouter(r *mux.Router) {
	r.Handle("/devices", jwtMiddleware.Handler(adapt(w.getDevsDataHandler, w.recoveryAdapter))).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", adapt(w.getDevDataHandler, w.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", adapt(w.getDevCfgHandler, w.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", adapt(w.patchDevCfgHandler, w.recoveryAdapter)).Methods(http.MethodPatch)
	// for Prometheus
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	// for auth + token
	r.Handle("/get-token", getTokenHandler).Methods(http.MethodGet)
}

func (w *Web) makeHTTPServer() *http.Server {
	r := mux.NewRouter()
	w.initRouter(r)
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

func (w *Web) makeHTTPToHTTPSRedirectServer() *http.Server {
	r := mux.NewRouter()
	r.Handle("/", adapt(w.redirectHandler, w.recoveryAdapter))
	return makeServerFromMux(r)
}

func (w *Web) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entity.EventPanic,
		}).Errorf("%w", err)
		panic("consul init error")
	}
	a := &consul.AgentServiceRegistration{
		Name: w.agentName,
		Port: w.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: w.ttl.String(),
		},
	}
	w.agent = c.Agent()
	if err := w.agent.ServiceRegister(a); err != nil {
		w.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entity.EventPanic,
		}).Errorf("%w", err)
		panic("consul init error")
	}
	go w.updateTTL(w.check)
}

func (w *Web) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (w *Web) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(w.ttl / 2)
	for range t.C {
		w.update(check)
	}
}

func (w *Web) update(check func() (bool, error)) {
	var h string
	ok, err := check()
	if !ok {
		w.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entity.EventUpdConsulStatus,
		}).Errorf("check has failed: %w", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		h = consul.HealthCritical
	} else {
		h = consul.HealthPassing
	}

	if err := w.agent.UpdateTTL("service:"+w.agentName, "", h); err != nil {
		w.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entity.EventUpdConsulStatus,
		}).Error(err)
	}
}

func (w *Web) listenTermination() {
	for {
		select {
		case <-w.ctrl.StopChan:
			w.terminate()
			return
		}
	}
}

func (w *Web) terminate() {
	defer func() {
		if r := recover(); r != nil {
			w.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": entity.EventPanic,
			}).Errorf("%w", r)
			w.ctrl.Terminate()
		}
	}()

	w.store.CloseConn()
	w.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entity.EventSVCShutdown,
	}).Infoln("svc is down")
	w.ctrl.Terminate()
}

// https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
type adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func adapt(hf http.HandlerFunc, adapters ...adapter) http.Handler {
	for _, adapter := range adapters {
		hf = adapter(hf)
	}
	return hf
}

func (w *Web) recoveryAdapter(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.log.WithFields(logrus.Fields{
					"func":  "recoveryAdapter",
					"event": entity.EventPanic,
				}).Errorf("%w", r)
				w.terminate()
			}
		}()
		h.ServeHTTP(wr, r)
	})
}
