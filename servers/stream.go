package servers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"fmt"
	"sync"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type StreamServer struct {
	LocalServer entities.Server
	Connections StreamConnMap
	PubSub      PubSub
	Upgrader    websocket.Upgrader
	Controller  entities.RoutinesController
	Storage     entities.Storage
}

type StreamConnMap struct {
	ConnChanCloseWS   chan *websocket.Conn
	StopCloseWS       chan string
	MacChan           chan string
	CloseMapCollector chan string
	MapConn           map[string]*ListConnection
	sync.Mutex
}

type PubSub struct {
	RoomIDForWSPubSub string
	StopSub           chan bool
	SubWSChannel      chan []string
}

type ListConnection struct {
	sync.Mutex
	Connections []*websocket.Conn
}

func NewWSConnections() *StreamConnMap {
	var (
		connChanCloseWS   = make(chan *websocket.Conn)
		stopCloseWS       = make(chan string)
		mapConn           = make(map[string]*ListConnection)
		MacChan           = make(chan string)
		CloseMapCollector = make(chan string)
	)
	return &StreamConnMap{
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

func NewStreamServer(s entities.Server, c entities.RoutinesController, st entities.Storage) *StreamServer {
	var (
		roomIDForDevWSPublish = "devWS"
		stopSub               = make(chan bool)
		subWSChannel          = make(chan []string)
	)

	return NewWSServer(s, *NewPubSub(roomIDForDevWSPublish, stopSub, subWSChannel),
		*NewWSConnections(), c, st)
}

// Return referenced address on the StreamServer with default Upgrader where:
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin: func(r *http.Request) bool {
//			return true
//	}
func NewWSServer(s entities.Server, pubSub PubSub, wsConnections StreamConnMap, c entities.RoutinesController,
	st entities.Storage) *StreamServer {
	u := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == s.Host+":"+fmt.Sprint(s.Port) {
				return true
			}
			return true
		},
	}

	return &StreamServer{
		LocalServer: s,
		Connections: wsConnections,
		PubSub:      pubSub,
		Upgrader:    u,
		Controller:  c,
		Storage:     st,
	}
}

func (ss *StreamServer) Recover() {
	if r := recover(); r != nil {
		ss.Connections.StopCloseWS <- ""
		ss.PubSub.StopSub <- false
		ss.Connections.CloseMapCollector <- ""
		ss.Controller.Close()
		log.Error("WebSocketServer Failed")
	}
}

func (ss *StreamServer) Run() {
	defer ss.Recover()

	conn, err := ss.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
	}
	defer conn.CloseConnection()

	go ss.Close()
	go ss.Subscribe(conn)
	go ss.Connections.MapCollector()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", ss.WSHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         ss.LocalServer.Host + ":" + fmt.Sprint(ss.LocalServer.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	go log.Fatal(srv.ListenAndServe())
}

func (ss *StreamServer) WSHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := ss.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	//http://..../device/type/name/mac
	ss.Connections.Lock()
	uri := strings.Split(r.URL.String(), "/")

	if _, ok := ss.Connections.MapConn[uri[2]]; !ok {
		ss.Connections.MapConn[uri[2]] = new(ListConnection)
	}
	ss.Connections.MapConn[uri[2]].Add(conn)
	ss.Connections.Unlock()

	log.Info(len(ss.Connections.MapConn))

}

/**
Delete connections in mapConn
*/
func (ss *StreamServer) Close() {
	defer ss.Recover()
	for {
		select {
		case connAddres := <-ss.Connections.ConnChanCloseWS:
			for key, val := range ss.Connections.MapConn {

				if ok := val.Remove(connAddres); ok {
					ss.Connections.MacChan <- key
					break
				}
			}
		case <-ss.Connections.StopCloseWS:
			log.Info("CloseConnection closed")
			return
		}
	}
}

/*
Listens changes in database. If they have, we will send to all websocket which working with them.
*/
func (ss *StreamServer) Subscribe(s entities.Storage) {
	defer ss.Recover()
	s.Subscribe(ss.PubSub.SubWSChannel, ss.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-ss.PubSub.SubWSChannel:
			if msg[0] == "message" {
				go ss.checkAndSendInfoToWSClient(msg)
			}
		case <-ss.PubSub.StopSub:
			log.Info("Subscribe closed")
			return
		}
	}
}

//We are check mac in our mapConnections.
// If we have mac in the map we will send message to all connections.
// Else we do nothing
func (ss *StreamServer) checkAndSendInfoToWSClient(msg []string) error {
	r := new(entities.Request)
	err := json.Unmarshal([]byte(msg[2]), &r)
	if err != nil {
		errors.Wrap(err, "Request unmarshalling has failed")
		return err
	}
	if _, ok := ss.Connections.MapConn[r.Meta.MAC]; ok {
		ss.sendInfoToWSClient(r.Meta.MAC, msg[2])
		return nil
	}
	return nil
}

//Send message to all connections which we have in map, and which pertain to mac
func (ss *StreamServer) sendInfoToWSClient(mac, msg string) {
	ss.Connections.MapConn[mac].Lock()
	for _, val := range ss.Connections.MapConn[mac].Connections {
		err := val.WriteMessage(1, []byte(msg))
		if err != nil {
			log.Errorf("Connection %v closed", val.RemoteAddr())
			go getToChannel(val, ss.Connections.ConnChanCloseWS)
		}
	}
	ss.Connections.MapConn[mac].Unlock()
}

func getToChannel(conn *websocket.Conn, connChanCloseWS chan *websocket.Conn) {
	connChanCloseWS <- conn
}

func (l *ListConnection) Add(conn *websocket.Conn) {
	l.Lock()
	l.Connections = append(l.Connections, conn)
	l.Unlock()

}

func (l *ListConnection) Remove(conn *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	position := 0
	for _, v := range l.Connections {
		if v == conn {
			l.Connections = append(l.Connections[:position], l.Connections[position+1:]...)
			log.Info("Web sockets connection deleted: ", conn.RemoteAddr())
			return true
		}
		position++
	}
	return false
}

func (m *StreamConnMap) Remove(mac string) {
	m.Lock()
	delete(m.MapConn, mac)
	m.Unlock()
}

func (m *StreamConnMap) MapCollector() {
	for {
		select {
		case mac := <-m.MacChan:
			m.Lock()
			if len(m.MapConn[mac].Connections) == 0 {
				delete(m.MapConn, mac)
				log.Info("REMOVE Connections --------------> ", len(m.MapConn), "  ", mac)

			}
			m.Unlock()

		case <-m.CloseMapCollector:
			return
		}
	}
}
