package services

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"sync"

	"context"

	"os"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type ConnList struct {
	sync.Mutex
	Conns []*websocket.Conn
}

func (l *ConnList) AddConn(conn *websocket.Conn) {
	l.Lock()
	l.Conns = append(l.Conns, conn)
	l.Unlock()
}

func (l *ConnList) RemoveConn(conn *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	for i, c := range l.Conns {
		if c == conn {
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

func (c *StreamConns) RemoveMACConn(mac string) {
	c.Lock()
	delete(c.ConnMap, mac)
	c.Unlock()
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

type Subscription struct {
	Subject string
	Channel chan []string
}

func NewSubscription(subject string, channel chan []string) *Subscription {
	return &Subscription{
		Subject: subject,
		Channel: channel,
	}
}

type StreamService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
	Conns      StreamConns
	DevDataSub Subscription
	Upgrader   websocket.Upgrader
}

func NewStreamService(s entities.Server, st entities.DevStorage, c entities.ServicesController, l *logrus.Logger) *StreamService {
	u := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == s.Host+":"+s.Port {
				return true
			}
			return true
		},
	}
	l.Out = os.Stdout

	return &StreamService{
		Server:     s,
		DevStorage: st,
		Controller: c,
		Log:        l,
		Conns:      *NewStreamConns(),
		DevDataSub: *NewSubscription(entities.DevDataChan, make(chan []string)),
		Upgrader:   u,
	}
}

func (s *StreamService) Run() {
	s.Log.Infof("StreamService is running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: Run(): panic(): %s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()

	go s.subscribe(ctx)
	go s.unsubscribe(ctx)
	go s.Conns.MapCollector(ctx)

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.devDataStreamHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.Server.Host + ":" + s.Server.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	s.Log.Fatal(srv.ListenAndServe())
}

func (s *StreamService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *StreamService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: terminate(): panic(): %s", r)
			s.terminate()
		}
	}()

	s.DevStorage.CloseConn()
	s.Log.Info("StreamService is down")
	s.Controller.Terminate()
}

func (s *StreamService) devDataStreamHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.Split(r.URL.String(), "/")

	s.Conns.Lock()
	defer s.Conns.Unlock()

	if _, ok := s.Conns.ConnMap[uri[2]]; !ok {
		s.Conns.ConnMap[uri[2]] = new(ConnList)
	}

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Errorf("StreamService: devDataStreamHandler(): Upgrade() has failed: %s", err)
		return
	}
	s.Conns.ConnMap[uri[2]].AddConn(conn)
}

func (s *StreamService) subscribe(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: subscribe(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("StreamService: subscribe(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	conn.Subscribe(s.DevDataSub.Channel, s.DevDataSub.Subject)

	for {
		select {
		case msg := <-s.DevDataSub.Channel:
			if msg[0] == "message" {
				go s.streamDevData(ctx, msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *StreamService) streamDevData(ctx context.Context, msgs []string) error {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: streamDevData(): panic(): %s", r)
			s.terminate()
		}
	}()

	var req entities.SaveDevDataRequest
	if err := json.Unmarshal([]byte(msgs[2]), &req); err != nil {
		errors.Wrap(err, "StreamService: streamDevData(): Request unmarshalling has failed")
		return err
	}

	if _, ok := s.Conns.ConnMap[req.Meta.MAC]; ok {
		s.Conns.ConnMap[req.Meta.MAC].Lock()
		for _, с := range s.Conns.ConnMap[req.Meta.MAC].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				if err := с.WriteMessage(1, []byte(msgs[2])); err != nil {
					errors.Errorf("StreamService: streamDevData(): connection %v has been closed", с.RemoteAddr())
					s.Conns.ConnChanCloseWS <- с
					return err
				}
			}
		}
		s.Conns.ConnMap[req.Meta.MAC].Unlock()
		return nil
	}
	return nil
}

func (s *StreamService) unsubscribe(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: unsubscribe(): panic(): %s", r)
			s.terminate()
		}
	}()

	for {
		select {
		case conn := <-s.Conns.ConnChanCloseWS:
			for k, v := range s.Conns.ConnMap {
				if ok := v.RemoveConn(conn); ok {
					s.Conns.MACChan <- k
					break
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
