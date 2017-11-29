package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"os"

	"github.com/giperboloid/centerms/entities"
)

type WebService struct {
	Server     entities.Server
	Storage    entities.Storage
	Ctrl       entities.ServiceController
	Log        *logrus.Logger
	PubSubject string
}

func NewWebService(srv entities.Server, st entities.Storage, c entities.ServiceController, l *logrus.Logger,
	subj string) *WebService {

	l.Out = os.Stdout
	return &WebService{
		Server:     srv,
		Storage:    st,
		Ctrl:       c,
		Log:        l,
		PubSubject: subj,
	}
}

func (s *WebService) Run() {
	s.Log.Infof("WebService    is running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("WebService: Run(): panic(): %s", r)
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
			s.Log.Errorf("WebService: terminate(): panic(): %s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.Infoln("WebService is down")
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
				s.Log.Errorf("WebService: panic(): %s", r)
				s.terminate()
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *WebService) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	data, err := conn.GetDevsData()
	if err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): devices data extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): []DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebService) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	id := mux.Vars(r)["id"]

	data, err := conn.GetDevData(id)
	if err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): DevData extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebService) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	id := mux.Vars(r)["id"]

	config, err := conn.GetDevConfig(id)
	if err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): DevConfig extraction has failed: %s", err)
		return
	}

	if _, err = w.Write(config.Data); err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): Write() has failed: %s", err)
		return
	}
}

func (s *WebService) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): storage connection hasn't been established: %s")
		return
	}
	defer conn.CloseConn()

	var config entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig decoding has failed: %s", err)
		return
	}

	id := r.URL.Query().Get("id")
	if err = conn.SetDevConfig(id, &config); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig setting has failed: %s", err)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig marshalling has failed: %s", err)
		return
	}

	if _, err = conn.Publish(s.PubSubject, b); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): stream() has failed: %s", err)
		return
	}
}
