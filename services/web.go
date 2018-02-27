package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/kostiamol/centerms/entities"
)

type WebService struct {
	Server  entities.Address
	Storage entities.Storager
	Ctrl    entities.ServiceController
	Log     *logrus.Entry
	PubChan string
}

func NewWebService(srv entities.Address, storage entities.Storager, ctrl entities.ServiceController, log *logrus.Entry,
	pubChan string) *WebService {

	return &WebService{
		Server:  srv,
		Storage: storage,
		Ctrl:    ctrl,
		Log:     log.WithFields(logrus.Fields{"service": "web"}),
		PubChan: pubChan,
	}
}

func (s *WebService) Run() {
	s.Log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)

	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	go s.listenTermination()

	r := mux.NewRouter()
	r.Handle("/devices", Adapt(s.getDevsDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", Adapt(s.getDevDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", Adapt(s.getDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", Adapt(s.patchDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodPatch)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.Server.Host + ":" + s.Server.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS", "PATCH"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	http.ListenAndServe(s.Server.Host+":"+s.Server.Port, handlers.CORS(allowedMethods, allowedHeaders)(r))
	s.Log.Fatal(srv.ListenAndServe())
}

func (s *WebService) listenTermination() {
	for {
		select {
		case <-s.Ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *WebService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": "panic",
			}).Errorf("%s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Infoln("WebService is down")
	s.Ctrl.Terminate()
}

// https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
type Adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func Adapt(hf http.HandlerFunc, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		hf = adapter(hf)
	}
	return hf
}

func (s *WebService) recoveryAdapter(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				s.Log.WithFields(logrus.Fields{
					"func":  "recoveryAdapter",
					"event": "panic",
				}).Errorf("%s", r)
				s.terminate()
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *WebService) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	data, err := conn.GetDevsData()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := mux.Vars(r)["id"]
	data, err := conn.GetDevData(id)
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := mux.Vars(r)["id"]
	config, err := conn.GetDevConfig(id)
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	if _, err = w.Write(config.Data); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	var config entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	id := mux.Vars(r)["id"]
	if err = conn.SetDevConfig(id, &config); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err = conn.Publish(b, s.PubChan); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}
