package servers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type WebServer struct {
	LocalServer entities.Server
	Controller  entities.RoutinesController
	Storage     entities.Storage
}

func NewWebServer(serv entities.Server, c entities.RoutinesController, s entities.Storage) *WebServer {
	return &WebServer{
		LocalServer: serv,
		Controller:  c,
		Storage:     s,
	}
}

func (s *WebServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			s.Controller.Close()
			log.Error("WebServer Failed")
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/devices", s.getDevicesHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/data", s.getDevDataHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config", s.getDevConfigHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config", s.patchDevConfigHandler).Methods(http.MethodPatch)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../view/")))

	port := fmt.Sprint(s.LocalServer.Port)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.LocalServer.Host + ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	//CORS provides Cross-Origin Resource Sharing middleware
	http.ListenAndServe(s.LocalServer.Host+":"+port, handlers.CORS()(r))

	go log.Fatal(srv.ListenAndServe())
}

func (s *WebServer) getDevicesHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	devs, err := conn.GetAllDevices()
	if err != nil {
		errors.Wrap(err, "devices info extraction has failed")
	}

	err = json.NewEncoder(w).Encode(devs)
	if err != nil {
		errors.Wrap(err, "[]DevData marshalling has failed")
	}
}

func (s *WebServer) getDevDataHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	devMeta := entities.DevMeta{
		Type: r.FormValue("type"),
		Name: r.FormValue("name"),
		MAC:  r.FormValue("mac"),
		IP:   "",
	}

	devID := "device:" + devMeta.Type + ":" + devMeta.Name + ":" + devMeta.MAC
	devParamsKey := devID + ":" + "params"

	deviceData, err := s.Storage.GetDevData(devParamsKey, devMeta)
	if err != nil {
		errors.Wrap(err, "dev data extraction has failed")
	}

	err = json.NewEncoder(w).Encode(deviceData)
	if err != nil {
		errors.Wrap(err, "DevData marshalling has failed")
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
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	configInfo, err := conn.GetKeyForConfig(devMeta.MAC) // key
	if err != nil {
		log.Errorln("key extraction has failed")
	}

	config, err := s.Storage.GetDevConfig(devMeta.Type, configInfo, devMeta.MAC)
	if err != nil {
		errors.Wrap(err, "dev config extraction has failed")
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

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	var config *entities.DevConfig
	configInfo := devMeta.MAC + ":" + "config"

	err = json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		errors.Wrap(err, "decoding has failed")
	}

	// validation
	s.Storage.SetDevConfig(devMeta.Type, configInfo, config)
	JSONConfig, _ := json.Marshal(config)
	s.Storage.Publish("configChan", JSONConfig)
}
