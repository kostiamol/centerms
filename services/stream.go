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

	"github.com/kostiamol/centerms/entities"
	"github.com/pkg/errors"
)

type connList struct {
	sync.RWMutex
	Conns []*websocket.Conn
}

func (l *connList) addConn(conn *websocket.Conn) {
	l.Lock()
	l.Conns = append(l.Conns, conn)
	l.Unlock()
}

func (l *connList) removeConn(conn *websocket.Conn) bool {
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

type streamConns struct {
	sync.RWMutex
	ClosedConns   chan *websocket.Conn
	MACConns      map[string]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		ClosedConns:   make(chan *websocket.Conn),
		MACConns:      make(map[string]*connList),
	}
}

func (c *streamConns) checkMACConns(mac string) {
	c.Lock()
	if len(c.MACConns[mac].Conns) == 0 {
		delete(c.MACConns, mac)
	}
	c.Unlock()
}

type StreamService struct {
	Server   entities.Server
	Storage  entities.Storage
	Ctrl     entities.ServiceController
	Log      *logrus.Logger
	Sub      entities.Subscription
	conns    streamConns
	upgrader websocket.Upgrader
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
		conns:   *newStreamConns(),
		Sub: entities.Subscription{
			Subject: subject,
			Channel: make(chan []string),
		},
		upgrader: u,
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

	s.conns.Lock()
	if _, ok := s.conns.MACConns[mac]; !ok {
		s.conns.MACConns[mac] = new(connList)
	}
	s.conns.Unlock()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Errorf("StreamService: addConnHandler(): Upgrade() has failed: %s", err)
		return
	}
	s.conns.MACConns[mac].addConn(conn)
	s.Log.Infof("StreamService: addConnHandler(): conn with %v has been added", conn.RemoteAddr())
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

	if _, ok := s.conns.MACConns[req.Meta.MAC]; ok {
		for _, conn := range s.conns.MACConns[req.Meta.MAC].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.conns.MACConns[req.Meta.MAC].Lock()
				err := conn.WriteMessage(1, []byte(message[2]))
				s.conns.MACConns[req.Meta.MAC].Unlock()
				if err != nil {
					s.Log.Infof("StreamService: stream(): conn with %v has been closed", conn.RemoteAddr())
					s.conns.ClosedConns <- conn
					return err
				}
			}
		}
		return nil
	}
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
		case conn := <-s.conns.ClosedConns:
			for mac, connList := range s.conns.MACConns {
				if ok := connList.removeConn(conn); ok {
					s.conns.checkMACConns(mac)
					break
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
