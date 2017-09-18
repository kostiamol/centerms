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
	"github.com/pkg/errors"
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
			s.Log.Error("WebServer: Run(): panic: ", r)
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

	r := mux.NewRouter()
	r.HandleFunc("/devices", s.getDevsDataHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/data", s.getDevDataHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config", s.getDevConfigHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config", s.patchDevConfigHandler).Methods(http.MethodPatch)
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

func (s *WebServer) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): storage connection hasn't been established")
		return
	}
	defer cn.CloseConn()

	ds, err := cn.GetDevsData()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): devices info extraction has failed")
	}

	err = json.NewEncoder(w).Encode(ds)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): []DevData encoding has failed")
	}
}

func (s *WebServer) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevDataHandler(): storage connection hasn't been established")
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
		errors.Wrap(err, "WebServer: getDevDataHandler(): DevData extraction has failed")
	}

	err = json.NewEncoder(w).Encode(dd)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevDataHandler(): DevData encoding has failed")
	}
}

func (s *WebServer) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Error("WebServer: getDevConfigHandler(): panic: ", r)
			s.gracefulHalt()
		}
	}()

	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevConfigHandler(): storage connection hasn't been established")
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
	}

	w.Write(dc.Data)
}

func (s *WebServer) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): storage connection hasn't been established")
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
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): DevConfig decoding has failed")
	}
	s.Log.Infof("Config after decoding: %s", dc)


	if err = cn.SetDevConfig(&dm, &dc); err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): DevConfig setting has failed")
	}

	b, err := json.Marshal(dc)
	if err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): DevConfig marshalling has failed")
	}

	if _, err = cn.Publish("devConfig", b); err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): Publish() has failed")
	}
	s.Log.Infof("WebServer: patchDevConfigHandler(): publish new config: %s for device %s", dc, dm)
}
