package servers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"fmt"
	"sync"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type StreamServer struct {
	Server     entities.Server
	Storage    entities.DevStore
	Controller entities.RoutinesController
	Log        *logrus.Logger
	Conns      StreamConnMap
	PubSub     PubSub
	Upgrader   websocket.Upgrader
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
	Conns []*websocket.Conn
}

func NewStreamServer(s entities.Server, st entities.DevStore, c entities.RoutinesController, l *logrus.Logger) *StreamServer {
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
		Server:     s,
		Storage:    st,
		Controller: c,
		Log:        l,
		Conns:      *NewWSConnections(),
		PubSub:     *NewPubSub("devWS", make(chan bool), make(chan []string)),
		Upgrader:   u,
	}
}

func NewWSConnections() *StreamConnMap {
	return &StreamConnMap{
		ConnChanCloseWS:   make(chan *websocket.Conn),
		StopCloseWS:       make(chan string),
		MapConn:           make(map[string]*ListConnection),
		MacChan:           make(chan string),
		CloseMapCollector: make(chan string),
	}
}

func NewPubSub(roomIDForWSPubSub string, stopSub chan bool, subWSChannel chan []string) *PubSub {
	return &PubSub{
		RoomIDForWSPubSub: roomIDForWSPubSub,
		StopSub:           stopSub,
		SubWSChannel:      subWSChannel,
	}
}

func (s *StreamServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			errors.New("StreamServer: Run(): panic leads to halt")
			s.gracefulHalt()
			s.Conns.StopCloseWS <- ""
			s.PubSub.StopSub <- false
			s.Conns.CloseMapCollector <- ""
			s.Controller.Close()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		errors.New("StreamServer: Run(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	go s.Close()
	go s.Subscribe(conn)
	go s.Conns.MapCollector()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.WSHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.Server.Host + ":" + fmt.Sprint(s.Server.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go s.Log.Fatal(srv.ListenAndServe())
}

func (s *StreamServer) gracefulHalt() {
	s.Storage.CloseConn()
}

func (s *StreamServer) WSHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		errors.Wrap(err, "StreamServer: WSHandler(): Upgrage() has failed")
		return
	}
	//http://..../device/type/name/mac
	s.Conns.Lock()
	uri := strings.Split(r.URL.String(), "/")

	if _, ok := s.Conns.MapConn[uri[2]]; !ok {
		s.Conns.MapConn[uri[2]] = new(ListConnection)
	}
	s.Conns.MapConn[uri[2]].AddConn(conn)
	s.Conns.Unlock()
}

/**
Delete connections in mapConn
*/
func (s *StreamServer) Close() {
	for {
		select {
		case connAddres := <-s.Conns.ConnChanCloseWS:
			for key, val := range s.Conns.MapConn {

				if ok := val.RemoveConn(connAddres); ok {
					s.Conns.MacChan <- key
					break
				}
			}
		case <-s.Conns.StopCloseWS:
			s.Log.Info("StreamServer: Close(): conn is down")
			return
		}
	}
}

/*
Listens changes in database. If they have, we will send to all websocket which working with them.
*/
func (s *StreamServer) Subscribe(st entities.DevStore) {
	st.Subscribe(s.PubSub.SubWSChannel, s.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-s.PubSub.SubWSChannel:
			if msg[0] == "message" {
				go s.checkAndSendInfoToWSClient(msg)
			}
		case <-s.PubSub.StopSub:
			s.Log.Info("StreamServer: Close(): subscribe is closed")
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
		errors.Wrap(err, "StreamServer: checkAndSendInfoToWSClient(): Request unmarshalling has failed")
		return err
	}
	if _, ok := s.Conns.MapConn[r.Meta.MAC]; ok {
		s.sendInfoToWSClient(r.Meta.MAC, msg[2])
		return nil
	}
	return nil
}

//Send message to all connections which we have in map, and which pertain to mac
func (s *StreamServer) sendInfoToWSClient(mac, msg string) {
	s.Conns.MapConn[mac].Lock()
	for _, val := range s.Conns.MapConn[mac].Conns {
		err := val.WriteMessage(1, []byte(msg))
		if err != nil {
			errors.Errorf("StreamServer: sendInfoToWSClient(): connection %v closed", val.RemoteAddr())
			go getToChannel(val, s.Conns.ConnChanCloseWS)
		}
	}
	s.Conns.MapConn[mac].Unlock()
}

func getToChannel(conn *websocket.Conn, connChanCloseWS chan *websocket.Conn) {
	connChanCloseWS <- conn
}

func (l *ListConnection) AddConn(conn *websocket.Conn) {
	l.Lock()
	l.Conns = append(l.Conns, conn)
	l.Unlock()
}

func (l *ListConnection) RemoveConn(conn *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	position := 0
	for _, v := range l.Conns {
		if v == conn {
			l.Conns = append(l.Conns[:position], l.Conns[position+1:]...)
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
			if len(m.MapConn[mac].Conns) == 0 {
				delete(m.MapConn, mac)
			}
			m.Unlock()

		case <-m.CloseMapCollector:
			return
		}
	}
}
