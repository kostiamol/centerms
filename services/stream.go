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
	ClosedConns   chan *websocket.Conn
	CheckMACConns chan string
	MACConns      map[string]*ConnList
}

func NewStreamConns() *StreamConns {
	return &StreamConns{
		ClosedConns:   make(chan *websocket.Conn),
		MACConns:      make(map[string]*ConnList),
		CheckMACConns: make(chan string),
	}
}

func (c *StreamConns) checkMACConns(ctx context.Context) {
	for {
		select {
		case mac := <-c.CheckMACConns:
			c.Lock()
			if len(c.MACConns[mac].Conns) == 0 {
				delete(c.MACConns, mac)
			}
			c.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

type StreamService struct {
	Server   entities.Server
	Storage  entities.Storage
	Ctrl     entities.ServiceController
	Log      *logrus.Logger
	Conns    StreamConns
	Sub      entities.Subscription
	Upgrader websocket.Upgrader
}

func NewStreamService(srv entities.Server, st entities.Storage, c entities.ServiceController, l *logrus.Logger,
	subject string) *StreamService {
	u := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == srv.Host+":"+srv.Port {
				return true
			}
			return true
		},
	}
	l.Out = os.Stdout

	return &StreamService{
		Server:  srv,
		Storage: st,
		Ctrl:    c,
		Log:     l,
		Conns:   *NewStreamConns(),
		Sub: entities.Subscription{
			Subject: subject,
			Channel: make(chan []string),
		},
		Upgrader: u,
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
	go s.listenPublications(ctx)
	go s.listenClosedConns(ctx)
	go s.Conns.checkMACConns(ctx)

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.addConnHandler)

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
		case <-s.Ctrl.StopChan:
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

	s.Storage.CloseConn()
	s.Log.Info("StreamService is down")
	s.Ctrl.Terminate()
}

func (s *StreamService) addConnHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: addConnHandler(): panic(): %s", r)
			s.terminate()
		}
	}()

	uri := strings.Split(r.URL.String(), "/")
	mac := uri[2]

	s.Conns.Lock()
	if _, ok := s.Conns.MACConns[mac]; !ok {
		s.Conns.MACConns[mac] = new(ConnList)
	}
	s.Conns.Unlock()

	s.Log.Info("conn req from 1")
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Errorf("StreamService: addConnHandler(): Upgrade() has failed: %s", err)
		return
	}
	s.Log.Info("conn req from 2")
	s.Conns.MACConns[mac].AddConn(conn)
	s.Log.Infof("StreamService(): addConnHandler(): conn with %v has been added", conn.RemoteAddr())
}

func (s *StreamService) listenPublications(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: listenPublications(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("StreamService: listenPublications(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()
	conn.Subscribe(s.Sub.Channel, s.Sub.Subject)

	for {
		select {
		case message := <-s.Sub.Channel:
			if message[0] == "message" {
				go s.stream(ctx, message)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *StreamService) stream(ctx context.Context, message []string) error {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: stream(): panic(): %s", r)
			s.terminate()
		}
	}()

	var req entities.SaveDevDataRequest
	if err := json.Unmarshal([]byte(message[2]), &req); err != nil {
		errors.Wrap(err, "StreamService: stream(): Request unmarshalling has failed")
		return err
	}

	s.Conns.Lock()
	if _, ok := s.Conns.MACConns[req.Meta.MAC]; ok {
		s.Conns.Unlock()
		s.Conns.MACConns[req.Meta.MAC].Lock()
		for _, conn := range s.Conns.MACConns[req.Meta.MAC].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				if err := conn.WriteMessage(1, []byte(message[2])); err != nil {
					s.Log.Infof("StreamService: stream(): conn with %v has been closed", conn.RemoteAddr())
					s.Conns.ClosedConns <- conn
					return err
				}
			}
		}
		s.Conns.MACConns[req.Meta.MAC].Unlock()
		return nil
	}
	s.Conns.Unlock()

	return nil
}

func (s *StreamService) listenClosedConns(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("StreamService: listenClosedConns(): panic(): %s", r)
			s.terminate()
		}
	}()

	for {
		select {
		case conn := <-s.Conns.ClosedConns:
			s.Conns.Lock()
			for mac, connList := range s.Conns.MACConns {
				if ok := connList.RemoveConn(conn); ok {
					s.Conns.CheckMACConns <- mac
					break
				}
			}
			s.Conns.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
