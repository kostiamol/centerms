package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// WebService is used to deal with user's queries from the web client (dashboard).
type WebService struct {
	addr      entities.Address
	storage   entities.Storager
	ctrl      entities.ServiceController
	log       *logrus.Entry
	pubChan   string
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewWebService creates and initializes a new instance of WebService.
func NewWebService(srv entities.Address, st entities.Storager, ctrl entities.ServiceController, log *logrus.Entry,
	pubChan, agentName string, ttl time.Duration) *WebService {

	return &WebService{
		addr:      srv,
		storage:   st,
		ctrl:      ctrl,
		log:       log.WithFields(logrus.Fields{"service": "web"}),
		pubChan:   pubChan,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination and queries from the web client.
func (s *WebService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	go s.listenTermination()

	s.runConsulAgent()

	r := mux.NewRouter()
	r.Handle("/devices", adapt(s.getDevsDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", adapt(s.getDevDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", adapt(s.getDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", adapt(s.patchDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodPatch)
	// for Prometheus
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.addr.Host + ":" + fmt.Sprint(s.addr.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS", "PATCH"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	http.ListenAndServe(s.addr.Host+":"+fmt.Sprint(s.addr.Port), handlers.CORS(allowedMethods, allowedHeaders)(r))
	s.log.Fatal(srv.ListenAndServe())
}

func (s *WebService) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	consulAgent := &consul.AgentServiceRegistration{
		Name: s.agentName,
		Port: s.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}
	s.agent = c.Agent()
	if err := s.agent.ServiceRegister(consulAgent); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	go s.updateTTL(s.check)
}

func (s *WebService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *WebService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *WebService) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := s.agent.UpdateTTL("service:"+s.agentName, "", health); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Error(err)
	}
}

func (s *WebService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *WebService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entities.EventSVCShutdown,
	}).Infoln("service is down")
	s.ctrl.Terminate()
}

// https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
type adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func adapt(hf http.HandlerFunc, adapters ...adapter) http.Handler {
	for _, adapter := range adapters {
		hf = adapter(hf)
	}
	return hf
}

func (s *WebService) recoveryAdapter(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				s.log.WithFields(logrus.Fields{
					"func":  "recoveryAdapter",
					"event": entities.EventPanic,
				}).Errorf("%s", r)
				s.terminate()
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *WebService) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	data, err := conn.GetDevsData()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entities.DevID(mux.Vars(r)["id"])
	data, err := conn.GetDevData(id)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entities.DevID(mux.Vars(r)["id"])
	config, err := conn.GetDevConfig(id)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	if _, err = w.Write(config.Data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	var config entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	id := entities.DevID(mux.Vars(r)["id"])
	if err = conn.SetDevConfig(entities.DevID(id), &config); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}
