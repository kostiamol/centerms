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
	sync.Mutex
	ConnChanCloseWS   chan *websocket.Conn
	StopCloseWS       chan string
	MacChan           chan string
	CloseMapCollector chan string
	MapConn           map[string]*ListConnection

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

func NewStreamServer(s entities.Server, c entities.RoutinesController, st entities.Storage) *StreamServer {
	var (
		roomIDForDevWSPublish = "devWS"
		stopSub               = make(chan bool)
		subWSChannel          = make(chan []string)
	)

	return NewWSServer(s, *NewPubSub(roomIDForDevWSPublish, stopSub, subWSChannel),
		*NewWSConnections(), c, st)
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

func (s *StreamServer) Recover() {
	if r := recover(); r != nil {
		s.Connections.StopCloseWS <- ""
		s.PubSub.StopSub <- false
		s.Connections.CloseMapCollector <- ""
		s.Controller.Close()
		log.Error("WebSocketServer Failed")
	}
}

func (s *StreamServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("StreamServer: Run(): panic leads to halt")
			s.gracefulHalt()
			s.Connections.StopCloseWS <- ""
			s.PubSub.StopSub <- false
			s.Connections.CloseMapCollector <- ""
			s.Controller.Close()
		}
	}()

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("StreamServer: Run(): db connection hasn't been established")
	}
	defer conn.CloseConnection()

	go s.Close()
	go s.Subscribe(conn)
	go s.Connections.MapCollector()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.WSHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.LocalServer.Host + ":" + fmt.Sprint(s.LocalServer.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	go log.Fatal(srv.ListenAndServe())
}

func (s *StreamServer) gracefulHalt() {
	s.Storage.CloseConnection()
}

func (s *StreamServer) WSHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	//http://..../device/type/name/mac
	s.Connections.Lock()
	uri := strings.Split(r.URL.String(), "/")

	if _, ok := s.Connections.MapConn[uri[2]]; !ok {
		s.Connections.MapConn[uri[2]] = new(ListConnection)
	}
	s.Connections.MapConn[uri[2]].Add(conn)
	s.Connections.Unlock()

	log.Info(len(s.Connections.MapConn))

}

/**
Delete connections in mapConn
*/
func (s *StreamServer) Close() {
	defer s.Recover()
	for {
		select {
		case connAddres := <-s.Connections.ConnChanCloseWS:
			for key, val := range s.Connections.MapConn {

				if ok := val.Remove(connAddres); ok {
					s.Connections.MacChan <- key
					break
				}
			}
		case <-s.Connections.StopCloseWS:
			log.Info("StreamServer: Close(): conn is down")
			return
		}
	}
}

/*
Listens changes in database. If they have, we will send to all websocket which working with them.
*/
func (s *StreamServer) Subscribe(st entities.Storage) {
	defer s.Recover()
	st.Subscribe(s.PubSub.SubWSChannel, s.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-s.PubSub.SubWSChannel:
			if msg[0] == "message" {
				go s.checkAndSendInfoToWSClient(msg)
			}
		case <-s.PubSub.StopSub:
			log.Info("Subscribe closed")
			return
		}
	}
}

//We are check mac in our mapConnections.
// If we have mac in the map we will send message to all connections.
// Else we do nothing
func (s *StreamServer) checkAndSendInfoToWSClient(msg []string) error {
	r := new(entities.Request)
	err := json.Unmarshal([]byte(msg[2]), &r)
	if err != nil {
		errors.Wrap(err, "Request unmarshalling has failed")
		return err
	}
	if _, ok := s.Connections.MapConn[r.Meta.MAC]; ok {
		s.sendInfoToWSClient(r.Meta.MAC, msg[2])
		return nil
	}
	return nil
}

//Send message to all connections which we have in map, and which pertain to mac
func (s *StreamServer) sendInfoToWSClient(mac, msg string) {
	s.Connections.MapConn[mac].Lock()
	for _, val := range s.Connections.MapConn[mac].Connections {
		err := val.WriteMessage(1, []byte(msg))
		if err != nil {
			log.Errorf("Connection %v closed", val.RemoteAddr())
			go getToChannel(val, s.Connections.ConnChanCloseWS)
		}
	}
	s.Connections.MapConn[mac].Unlock()
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
