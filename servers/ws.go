package servers

import (
	"time"
	"net/http"
	"encoding/json"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/giperboloid/centerms/entities"
	. "github.com/giperboloid/centerms/sys"
	"fmt"
	"github.com/giperboloid/centerms/db"
)

type WSServer struct {
	LocalServer entities.Server
	Connections WSConnectionsMap
	PubSub      PubSub
	Upgrader    websocket.Upgrader
	Controller  entities.RoutinesController
	DbClient	db.Client
}

func NewWSConnections() *WSConnectionsMap {
	var (
		connChanCloseWS   = make(chan *websocket.Conn)
		stopCloseWS       = make(chan string)
		mapConn           = make(map[string]*ListConnection)
		MacChan           = make(chan string)
		CloseMapCollector = make(chan string)
	)
	return &WSConnectionsMap{
		ConnChanCloseWS:   connChanCloseWS,
		StopCloseWS:       stopCloseWS,
		MapConn:           mapConn,
		MacChan:           MacChan,
		CloseMapCollector: CloseMapCollector,
	}
}

func NewPubSub(roomIDForWSPubSub string, stopSub chan bool, subWSChannel chan []string) *PubSub {
	return &PubSub{
		RoomIDForWSPubSub: roomIDForWSPubSub,
		StopSub:           stopSub,
		SubWSChannel:      subWSChannel,
	}
}

func NewWebSocketServer(ws entities.Server, controller entities.RoutinesController, dbClient db.Client) *WSServer {
	var (
		roomIDForDevWSPublish = "devWS"
		stopSub               = make(chan bool)
		subWSChannel          = make(chan []string)
	)

	return NewWSServer(ws, *NewPubSub(roomIDForDevWSPublish, stopSub, subWSChannel),
		*NewWSConnections(), controller, dbClient)
}

// Return referenced address on the WSServer with default Upgrader where:
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin: func(r *http.Request) bool {
//			return true
//	}
func NewWSServer(ws entities.Server, pubSub PubSub, wsConnections WSConnectionsMap, controller entities.RoutinesController, dbClient db.Client) *WSServer {
	var (
		upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if r.Host == ws.Host+":"+fmt.Sprint(ws.Port) {
					return true
				}
				return true
			},
		}
	)
	return &WSServer{
		LocalServer: ws,
		Connections: wsConnections,
		PubSub:      pubSub,
		Upgrader:    upgrader,
		Controller:  controller,
		DbClient: dbClient,
	}
}

func (server *WSServer) wsRecover(){
		if r := recover(); r != nil {
			server.Connections.StopCloseWS <- ""
			server.PubSub.StopSub <- false
			server.Connections.CloseMapCollector <- ""
			server.Controller.Close()
			log.Error("WebSocketServer Failed")
		}

}

//http web socket connection
func (server *WSServer) Run() {
	defer  server.wsRecover()

	dbClient := server.DbClient.NewDBConnection()

	go server.Close()
	go server.Subscribe(dbClient)
	go server.Connections.MapCollector()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", server.WSHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         server.LocalServer.Host + ":" + fmt.Sprint(server.LocalServer.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	go log.Fatal(srv.ListenAndServe())
}

func (server *WSServer) WSHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := server.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	//http://..../device/type/name/mac
	server.Connections.Lock()
	uri := strings.Split(r.URL.String(), "/")

	if _, ok := server.Connections.MapConn[uri[2]]; !ok {
		server.Connections.MapConn[uri[2]] = new(ListConnection)
	}
	server.Connections.MapConn[uri[2]].Add(conn)
	server.Connections.Unlock()

	log.Info(len(server.Connections.MapConn))

}

/**
Delete connections in mapConn
*/
func (server *WSServer) Close() {
	defer  server.wsRecover()
	for {
		select {
		case connAddres := <-server.Connections.ConnChanCloseWS:
			for key, val := range server.Connections.MapConn {

				if ok := val.Remove(connAddres); ok {
					server.Connections.MacChan <- key
					break
				}
			}
		case <-server.Connections.StopCloseWS:
			log.Info("Close closed")
			return
		}
	}
}

/*
Listens changes in database. If they have, we will send to all websocket which working with them.
*/
func (server *WSServer) Subscribe(dbWorker db.Client) {
	defer  server.wsRecover()
	dbWorker.Subscribe(server.PubSub.SubWSChannel, server.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-server.PubSub.SubWSChannel:
			if msg[0] == "message" {
				go server.checkAndSendInfoToWSClient(msg)
			}
		case <-server.PubSub.StopSub:
			log.Info("Subscribe closed")
			return
		}
	}
}

//We are check mac in our mapConnections.
// If we have mac in the map we will send message to all connections.
// Else we do nothing
func (server *WSServer) checkAndSendInfoToWSClient(msg []string) {
	r := new(entities.Request)
	err := json.Unmarshal([]byte(msg[2]), &r)
	if CheckError("checkAndSendInfoToWSClient", err) != nil {
		return
	}
	if _, ok := server.Connections.MapConn[r.Meta.MAC]; ok {
		server.sendInfoToWSClient(r.Meta.MAC, msg[2])
		return
	}
	log.Infof("mapConn dont have this MAC: %v. Len map is %v", r.Meta.MAC, len(server.Connections.MapConn))
}

//Send message to all connections which we have in map, and which pertain to mac
func (server *WSServer) sendInfoToWSClient(mac, message string) {
	server.Connections.MapConn[mac].Lock()
	for _, val := range server.Connections.MapConn[mac].Connections {
		err := val.WriteMessage(1, []byte(message))
		if err != nil {
			log.Errorf("Connection %v closed", val.RemoteAddr())
			go getToChannel(val, server.Connections.ConnChanCloseWS)
		}
	}
	server.Connections.MapConn[mac].Unlock()
}

func getToChannel(conn *websocket.Conn, connChanCloseWS chan *websocket.Conn) {
	connChanCloseWS <- conn
}
