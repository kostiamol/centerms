package servers

import (
	"fmt"
	"net/http"
	"time"
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/devices"
	. "github.com/giperboloid/centerms/sys"
	"github.com/giperboloid/centerms/db"
)

type HTTPServer struct {
	LocalServer entities.Server
	Controller  entities.RoutinesController
	DbClient    db.Client
}

func NewHTTPServer(local entities.Server, controller entities.RoutinesController, dbClient db.Client) *HTTPServer {
	return &HTTPServer{
		LocalServer: local,
		Controller:  controller,

		DbClient: dbClient,
	}
}

func (server *HTTPServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			server.Controller.Close()
			log.Error("HTTPServer Failed")
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/devices", 				server.getDevicesHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/data", 	server.getDevDataHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config",	server.getDevConfigHandler).Methods(http.MethodGet)
	r.HandleFunc("/devices/{id}/config", 	server.patchDevConfigHandler).Methods(http.MethodPatch)

	r.HandleFunc("/devices/washer", 		server.postDevConfigHandler).Methods(http.MethodPost)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../view/")))

	port := fmt.Sprint(server.LocalServer.Port)

	srv := &http.Server{
		Handler:      r,
		Addr:         server.LocalServer.Host + ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	//CORS provides Cross-Origin Resource Sharing middleware
	http.ListenAndServe(server.LocalServer.Host+":"+port, handlers.CORS()(r))

	go log.Fatal(srv.ListenAndServe())
}

func (server *HTTPServer) getDevicesHandler(w http.ResponseWriter, r *http.Request) {
	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	devices := dbClient.GetAllDevices()
	err := json.NewEncoder(w).Encode(devices)
	CheckError("getDevicesHandler JSON enc", err)
}

func (server *HTTPServer) getDevDataHandler(w http.ResponseWriter, r *http.Request) {

	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	log.Info(dbClient.GetClient())

	devMeta := entities.DevMeta{r.FormValue("type"), r.FormValue("name"), r.FormValue("mac"), ""}

	devID := "device:" + devMeta.Type + ":" + devMeta.Name + ":" + devMeta.MAC
	devParamsKey := devID + ":" + "params"

	device := IdentifyDevice(devMeta.Type)
	deviceData := device.GetDevData(devParamsKey, devMeta, dbClient)

	err := json.NewEncoder(w).Encode(deviceData)
	CheckError("getDevDataHandler JSON enc", err)
}

func (server *HTTPServer) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	devMeta := entities.DevMeta{r.FormValue("type"), r.FormValue("name"), r.FormValue("mac"), ""}
	flag, err := entities.ValidateDevMeta(devMeta)
	if !flag {
		log.Errorf("getDevConfigHandler. %v. Exit the method %v", err, devMeta.MAC)
		return
	}

	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	configInfo := dbClient.GetKeyForConfig(devMeta.MAC) // key

	var device devices.DevConfigDriver = IdentifyDevice(devMeta.Type)

	if device == nil {
		http.Error(w, "This type is not found", 400)
		return
	}
	config := device.GetDevConfig(configInfo, devMeta.MAC, dbClient)
	w.Write(config.Data)
}

func (server *HTTPServer) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {

	devMeta := entities.DevMeta{r.FormValue("type"), r.FormValue("name"), r.FormValue("mac"), ""}
	_, err := entities.ValidateDevMeta(devMeta)
	if err != nil {
		log.Error(err)
		return
	}

	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	device := IdentifyDevice(devMeta.Type)
	if device == nil {
		http.Error(w, "This type is not found", 400)
		return
	}
	device.PatchDevConfigHandlerHTTP(w, r, devMeta, dbClient)
}

func (server *HTTPServer) postDevConfigHandler(w http.ResponseWriter, r *http.Request) {

	var config entities.DevConfig
	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	json.NewDecoder(r.Body).Decode(&config)

	configInfo := config.MAC + ":" + "config"

	flag := entities.ValidateMAC(config.MAC)
	if !flag {
		log.Errorf("postDevConfigHandler. %v. Exit the method. Mac Invalid")
		json.NewEncoder(w).Encode(http.StatusOK)
		return
	}
	device := IdentifyDevice("washer")
	device.SetDevConfig(configInfo, &config, dbClient)
}


func IdentifyDevice(devType string)(devices.Driver){
	var device devices.Driver

	switch devType {
	case "fridge":
		device = &devices.Fridge{}
	case "washer":
		device = &devices.Washer{}
	default:
		log.Println("Device request: unknown device type")
		return nil
	}
	return device
}

