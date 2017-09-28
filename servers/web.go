package servers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"os"

	"github.com/giperboloid/centerms/entities"
)

type WebServer struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServersController
	Log        *logrus.Logger
}

func NewWebServer(s entities.Server, ds entities.DevStorage, sc entities.ServersController, l *logrus.Logger) *WebServer {
	l.Out = os.Stdout

	return &WebServer{
		Server:     s,
		DevStorage: ds,
		Controller: sc,
		Log:        l,
	}
}

func (s *WebServer) Run() {
	s.Log.Infof("WebServer has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("WebServer: Run(): panic: %s", r)
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

	r := mux.NewRouter()
	r.Handle("/devices", s.recoverWrap(s.getDevsDataHandler)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/data", s.recoverWrap(s.getDevDataHandler)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", s.recoverWrap(s.getDevConfigHandler)).Methods(http.MethodGet)
	r.Handle("/devices/{id}/config", s.recoverWrap(s.patchDevConfigHandler)).Methods(http.MethodPatch)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./view/")))

	port := fmt.Sprint(s.Server.Port)
	srv := &http.Server{
		Handler:      r,
		Addr:         s.Server.Host + ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	http.ListenAndServe(s.Server.Host+":"+port, handlers.CORS()(r))
	go s.Log.Fatal(srv.ListenAndServe())
}

func (s *WebServer) handleTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.gracefulHalt()
			return
		}
	}
}

func (s *WebServer) gracefulHalt() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("WebServer has shut down")
	s.Controller.Terminate()
}

func (s *WebServer) recoverWrap(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				s.Log.Errorf("WebServer: panic: %s", r)
				s.gracefulHalt()
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *WebServer) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebServer: getDevicesHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer cn.CloseConn()

	ds, err := cn.GetDevsData()
	if err != nil {
		s.Log.Errorf("WebServer: getDevicesHandler(): devices data extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(ds); err != nil {
		s.Log.Errorf("WebServer: getDevicesHandler(): []DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebServer) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebServer: getDevDataHandler(): storage connection hasn't been established: %s", err)
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
		s.Log.Errorf("WebServer: getDevDataHandler(): DevData extraction has failed: %s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(dd); err != nil {
		s.Log.Errorf("WebServer: getDevDataHandler(): DevData encoding has failed: %s", err)
		return
	}
}

func (s *WebServer) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebServer: getDevConfigHandler(): storage connection hasn't been established: %s", err)
		return
	}
	defer cn.CloseConn()

	dm := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
	}

	dc, err := cn.GetDevConfig(&dm)
	if err != nil {
		s.Log.Errorf("WebServer: getDevConfigHandler(): DevConfig extraction has failed: %s", err)
		return
	}

	if _, err = w.Write(dc.Data); err != nil {
		s.Log.Errorf("WebServer: getDevConfigHandler(): Write() has failed: %s", err)
		return
	}
}

func (s *WebServer) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("WebServer: patchDevConfigHandler(): storage connection hasn't been established: %s")
		return
	}
	defer cn.CloseConn()

	dm := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
	}

	var dc entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&dc); err != nil {
		s.Log.Errorf("WebServer: patchDevConfigHandler(): DevConfig decoding has failed: %s", err)
		return
	}

	if err = cn.SetDevConfig(&dm, &dc); err != nil {
		s.Log.Errorf("WebServer: patchDevConfigHandler(): DevConfig setting has failed: %s", err)
		return
	}

	b, err := json.Marshal(dc)
	if err != nil {
		s.Log.Errorf("WebServer: patchDevConfigHandler(): DevConfig marshalling has failed: %s", err)
		return
	}

	if _, err = cn.Publish(entities.DevConfigChan, b); err != nil {
		s.Log.Errorf("WebServer: patchDevConfigHandler(): Publish() has failed: %s", err)
		return
	}
	//s.Log.Infof("publish config patch: %s for device with MAC [%s]", dc.Data, dc.MAC)
}
