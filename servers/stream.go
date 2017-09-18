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

	"context"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type ConnList struct {
	sync.Mutex
	Conns []*websocket.Conn
}

func (l *ConnList) AddConn(conn *websocket.Conn) {
	l.Lock()
	defer l.Unlock()
	l.Conns = append(l.Conns, conn)
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

type StreamConns struct {
	sync.Mutex
	ConnChanCloseWS chan *websocket.Conn
	MACChan         chan string
	ConnMap         map[string]*ConnList
}

func NewStreamConns() *StreamConns {
	return &StreamConns{
		ConnChanCloseWS: make(chan *websocket.Conn),
		ConnMap:         make(map[string]*ConnList),
		MACChan:         make(chan string),
	}
}

func (c *StreamConns) RemoveConn(mac string) {
	c.Lock()
	defer c.Unlock()
	delete(c.ConnMap, mac)
}

func (c *StreamConns) MapCollector(ctx context.Context) {
	for {
		select {
		case mac := <-c.MACChan:
			c.Lock()
			if len(c.ConnMap[mac].Conns) == 0 {
				delete(c.ConnMap, mac)
			}
			c.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

type PubSub struct {
	SubWSChann        chan []string
	RoomIDForWSPubSub string
}

func NewPubSub(roomIDForWSPubSub string, stopSub chan bool, subWSChannel chan []string) *PubSub {
	return &PubSub{
		RoomIDForWSPubSub: roomIDForWSPubSub,
		SubWSChann:        subWSChannel,
	}
}

type StreamServer struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServersController
	Log        *logrus.Logger
	Conns      StreamConns
	PubSub     PubSub
	Upgrader   websocket.Upgrader
}

func NewStreamServer(s entities.Server, st entities.DevStorage, c entities.ServersController, l *logrus.Logger) *StreamServer {
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
		DevStorage: st,
		Controller: c,
		Log:        l,
		Conns:      *NewStreamConns(),
		PubSub:     *NewPubSub("devWS", make(chan bool), make(chan []string)),
		Upgrader:   u,
	}
}

func (s *StreamServer) Run() {
	s.Log.Infof("StreamServer has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			errors.New("StreamServer: Run(): panic leads to halt")
			cancel()
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.New("StreamServer: Run(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	go s.Subscribe(ctx, conn)
	go s.Unsubscribe(ctx)
	go s.Conns.MapCollector(ctx)

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

func (s *StreamServer) handleTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.gracefulHalt()
			return
		}
	}
}

func (s *StreamServer) gracefulHalt() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("StreamServer has shut down")
	s.Controller.Terminate()
}

func (s *StreamServer) devDataStreamHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.Split(r.URL.String(), "/")

	s.Conns.Lock()
	defer s.Conns.Unlock()
	if _, ok := s.Conns.ConnMap[uri[2]]; !ok {
		s.Conns.ConnMap[uri[2]] = new(ConnList)
	}

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		errors.Wrap(err, "StreamServer: devDataStreamHandler(): Upgrage() has failed")
		return
	}

	s.Conns.ConnMap[uri[2]].AddConn(conn)
}

func (s *StreamServer) Subscribe(ctx context.Context, st entities.DevStorage) {
	st.Subscribe(s.PubSub.SubWSChann, s.PubSub.RoomIDForWSPubSub)
	for {
		select {
		case msg := <-s.PubSub.SubWSChann:
			if msg[0] == "message" {
				go s.Publish(msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *StreamServer) Unsubscribe(ctx context.Context) {
	for {
		select {
		case connAddr := <-s.Conns.ConnChanCloseWS:
			for k, v := range s.Conns.ConnMap {
				if ok := v.RemoveConn(connAddr); ok {
					s.Conns.MACChan <- k
					break
				}
			}
		case <-ctx.Done():
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
				s.Conns.ConnChanCloseWS <- val
			}
		}
		s.Conns.ConnMap[req.Meta.MAC].Unlock()
		return nil
	}
	return nil
}
