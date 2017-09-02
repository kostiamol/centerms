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
	Storage    entities.Storage
	Controller entities.RoutinesController
	Log        *logrus.Logger
}

func NewWebServer(s entities.Server, st entities.Storage, c entities.RoutinesController, l *logrus.Logger) *WebServer {
	l.Out = os.Stdout
	return &WebServer{
		Server:     s,
		Storage:    st,
		Controller: c,
		Log:        l,
	}
}

func (s *WebServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			errors.New("WebServer: Run(): panic leads to halt")
			s.gracefulHalt()
			s.Controller.Close()
		}
	}()

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

func (s *WebServer) gracefulHalt() {
	s.Storage.CloseConnection()
}

func (s *WebServer) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConnection()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	devs, err := conn.GetDevsData()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): devices info extraction has failed")
	}

	err = json.NewEncoder(w).Encode(devs)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevicesHandler(): []DevData encoding has failed")
	}
}

func (s *WebServer) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	devMeta := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
		IP:   "",
	}

	devID := "device:" + devMeta.Type + ":" + devMeta.Name + ":" + devMeta.MAC
	devParamsKey := devID + ":" + "params"

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevDataHandler(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	devData, err := conn.GetDevData(devParamsKey, &devMeta)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevDataHandler(): DevData extraction has failed")
	}

	err = json.NewEncoder(w).Encode(devData)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevDataHandler(): DevData encoding has failed")
	}
}

func (s *WebServer) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	devMeta := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
		IP:   "",
	}

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevConfigHandler(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	configInfo, err := conn.GetConfigKey(devMeta.MAC)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevConfigHandler(): config key extraction has failed")
	}

	config, err := conn.GetDevConfig(devMeta.Type, configInfo, devMeta.MAC)
	if err != nil {
		errors.Wrap(err, "WebServer: getDevConfigHandler(): DevConfig extraction has failed")
	}
	w.Write(config.Data)
}

func (s *WebServer) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	devMeta := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
		IP:   "",
	}

	var config *entities.DevConfig
	configInfo := devMeta.MAC + ":" + "config"

	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): DevConfig decoding has failed")
	}

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		errors.Wrap(err, "WebServer: getDevConfigHandler(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	// validation must be implemented
	conn.SetDevConfig(devMeta.Type, configInfo, config)
	JSONConfig, err := json.Marshal(config)
	if err != nil {
		errors.Wrap(err, "WebServer: patchDevConfigHandler(): DevConfig marshalling has failed")
	}

	conn.Publish("configChan", JSONConfig)
}
