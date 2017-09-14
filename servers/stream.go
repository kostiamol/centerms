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
	Storage    entities.DevStorage
	Controller entities.RoutinesController
	Log        *logrus.Logger
	Conns      ConnMap
	PubSub     PubSub
	Upgrader   websocket.Upgrader
}

type ConnMap struct {
	sync.Mutex
	ConnChanCloseWS   chan *websocket.Conn
	StopCloseWS       chan string
	MACChan           chan string
	CloseMapCollector chan string
	ConnMap           map[string]*ConnList
}

type PubSub struct {
	StopSub           chan bool
	SubWSChann        chan []string
	RoomIDForWSPubSub string
}

type ConnList struct {
	sync.Mutex
	Conns []*websocket.Conn
}

func NewStreamServer(s entities.Server, st entities.DevStorage, c entities.RoutinesController, l *logrus.Logger) *StreamServer {
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
		Conns:      *NewWSConns(),
		PubSub:     *NewPubSub("devWS", make(chan bool), make(chan []string)),
		Upgrader:   u,
	}
}

func NewWSConns() *ConnMap {
	return &ConnMap{
		ConnChanCloseWS:   make(chan *websocket.Conn),
		StopCloseWS:       make(chan string),
		ConnMap:           make(map[string]*ConnList),
		MACChan:           make(chan string),
		CloseMapCollector: make(chan string),
	}
}

func NewPubSub(roomIDForWSPubSub string, stopSub chan bool, subWSChannel chan []string) *PubSub {
	return &PubSub{
		RoomIDForWSPubSub: roomIDForWSPubSub,
		StopSub:           stopSub,
		SubWSChann:        subWSChannel,
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

	go s.Subscribe(conn)
	go s.Unsubscribe()
	go s.Conns.MapCollector()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.devDataStreamHandler)

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

func (s *StreamServer) devDataStreamHandler(w http.ResponseWriter, r *http.Request) {
	cnn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		errors.Wrap(err, "StreamServer: devDataStreamHandler(): Upgrage() has failed")
		return
	}

	s.Conns.Lock()
	uri := strings.Split(r.URL.String(), "/")

	if _, ok := s.Conns.ConnMap[uri[2]]; !ok {
		s.Conns.ConnMap[uri[2]] = new(ConnList)
	}
	s.Conns.ConnMap[uri[2]].AddConn(cnn)
	s.Conns.Unlock()
}

func (s *StreamServer) Subscribe(st entities.DevStorage) {
	st.Subscribe(s.PubSub.SubWSChann, s.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-s.PubSub.SubWSChann:
			if msg[0] == "message" {
				go s.Publish(msg)
			}
		case <-s.PubSub.StopSub:
			return
		}
	}
}

func (s *StreamServer) Unsubscribe() {
	for {
		select {
		case connAddr := <-s.Conns.ConnChanCloseWS:
			for k, v := range s.Conns.ConnMap {
				if ok := v.RemoveConn(connAddr); ok {
					s.Conns.MACChan <- k
					break
				}
			}
		case <-s.Conns.StopCloseWS:
			return
		}
	}
}

func (s *StreamServer) Publish(msgs []string) error {
	var req entities.Request
	err := json.Unmarshal([]byte(msgs[2]), &req)
	if err != nil {
		errors.Wrap(err, "StreamServer: Publish(): Request unmarshalling has failed")
		return err
	}

	if _, ok := s.Conns.ConnMap[req.Meta.MAC]; ok {
		s.Conns.ConnMap[req.Meta.MAC].Lock()
		for _, val := range s.Conns.ConnMap[req.Meta.MAC].Conns {
			err := val.WriteMessage(1, []byte(msgs[2]))
			if err != nil {
				errors.Errorf("StreamServer: Publish(): connection %v has been closed", val.RemoteAddr())
				go getToChannel(val, s.Conns.ConnChanCloseWS)
			}
		}
		s.Conns.ConnMap[req.Meta.MAC].Unlock()
		return nil
	}
	return nil
}

func getToChannel(conn *websocket.Conn, connChanCloseWS chan *websocket.Conn) {
	connChanCloseWS <- conn
}

func (l *ConnList) AddConn(conn *websocket.Conn) {
	l.Lock()
	l.Conns = append(l.Conns, conn)
	l.Unlock()
}

func (l *ConnList) RemoveConn(conn *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	for i, v := range l.Conns {
		if v == conn {
			l.Conns = append(l.Conns[:i], l.Conns[i+1:]...)
			return true
		}
	}
	return false
}

func (m *ConnMap) RemoveConn(mac string) {
	m.Lock()
	delete(m.ConnMap, mac)
	m.Unlock()
}

func (m *ConnMap) MapCollector() {
	for {
		select {
		case mac := <-m.MACChan:
			m.Lock()
			if len(m.ConnMap[mac].Conns) == 0 {
				delete(m.ConnMap, mac)
			}
			m.Unlock()
		case <-m.CloseMapCollector:
			return
		}
	}
}
