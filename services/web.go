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
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
}

func NewWebService(s entities.Server, ds entities.DevStorage, sc entities.ServicesController, l *logrus.Logger) *WebService {
	l.Out = os.Stdout
	return &WebService{
		Server:     s,
		DevStorage: ds,
		Controller: sc,
		Log:        l,
	}
}

func (s *WebService) Run() {
	s.Log.Infof("WebService    is running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("WebService: Run(): panic(): %s", r)
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	r := mux.NewRouter()
	r.Handle("/devices", Adapt(s.getDevsDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", Adapt(s.getDevDataHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", Adapt(s.getDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", Adapt(s.patchDevConfigHandler, s.recoveryAdapter)).Methods(http.MethodPatch)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./view/")))

	srv := &http.Server{
		Handler:      r,
		Addr:         s.Server.Host + ":" + s.Server.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	http.ListenAndServe(s.Server.Host+":"+s.Server.Port, handlers.CORS()(r))
	go s.Log.Fatal(srv.ListenAndServe())
}

func (s *WebService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *WebService) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("WebService has shut down")
	s.Controller.Terminate()
}

// https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
type Adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func Adapt(h http.HandlerFunc, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

func (s *WebService) recoveryAdapter(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				s.Log.Errorf("WebService: panic(): %s", r)
				s.handleTermination()
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *WebService) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer cn.CloseConn()

	ds, err := cn.GetDevsData()
	if err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): devices data extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(ds); err != nil {
		s.Log.Errorf("WebService: getDevicesHandler(): []DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebService) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer cn.CloseConn()

	dm := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
	}

	dd, err := cn.GetDevData(&dm)
	if err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): DevData extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(dd); err != nil {
		s.Log.Errorf("WebService: getDevDataHandler(): DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebService) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	dm := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
	}

	dc, err := conn.GetDevConfig(&dm)
	if err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): DevConfig extraction has failed: %s", err)
		return
	}

	if _, err = w.Write(dc.Data); err != nil {
		s.Log.Errorf("WebService: getDevConfigHandler(): Write() has failed: %s", err)
		return
	}
}

func (s *WebService) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): storage connection hasn't been established: %s")
		return
	}
	defer conn.CloseConn()

	dm := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
	}

	var dc entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&dc); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig decoding has failed: %s", err)
		return
	}

	if err = conn.SetDevConfig(&dm, &dc); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig setting has failed: %s", err)
		return
	}

	b, err := json.Marshal(dc)
	if err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): DevConfig marshalling has failed: %s", err)
		return
	}

	if _, err = conn.Publish(entities.DevConfigChan, b); err != nil {
		s.Log.Errorf("WebService: patchDevConfigHandler(): Publish() has failed: %s", err)
		return
	}
}
