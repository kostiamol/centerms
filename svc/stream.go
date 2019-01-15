package svc

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kostiamol/centerms/cfg"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"sync"

	"context"

	"fmt"
)

// StreamService is used to deal with streaming of data from the device to web client (dashboard).
type StreamService struct {
	port     int
	store    Storer
	ctrl     Ctrl
	log      *logrus.Entry
	sub      subscription
	conns    streamConns
	upgrader websocket.Upgrader
}

// StreamServiceCfg is used to initialize an instance of StreamService.
type StreamServiceCfg struct {
	Port    int
	Store   Storer
	Ctrl    Ctrl
	Log     *logrus.Entry
	PubChan string
}

// NewStreamService creates and initializes a new instance of StreamService service.
func NewStreamService(c *StreamServiceCfg) *StreamService {
	return &StreamService{
		port:  c.Port,
		store: c.Store,
		ctrl:  c.Ctrl,
		log:   c.Log.WithFields(logrus.Fields{"component": "stream"}),
		conns: *newStreamConns(),
		sub: subscription{
			ChanName: c.PubChan,
			Chan:     make(chan []byte),
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Run launches the service by running goroutines for listening the service termination, new device data,
// closed web client connections and publishing new device data to web clients with open connections.
func (s *StreamService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("is running on port: [%d]", s.port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
	go s.listenPubs(ctx)
	go s.listenClosedConns(ctx)

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.addConnHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + fmt.Sprint(s.port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	s.log.Fatal(srv.ListenAndServe())
}

func (s *StreamService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *StreamService) terminate() {
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": cfg.EventSVCShutdown,
	}).Info("svc is down")
	s.ctrl.Terminate()
}

func (s *StreamService) addConnHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "addConnHandler",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	uri := strings.Split(r.URL.String(), "/")
	id := DevID(uri[2])

	s.conns.Lock()
	if _, ok := s.conns.idConns[id]; !ok {
		s.conns.idConns[id] = new(connList)
	}
	s.conns.Unlock()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "addConnHandler",
		}).Errorf("Upgrade() failed %s", err)
		return
	}
	s.conns.idConns[id].addConn(conn)
	s.log.WithFields(logrus.Fields{
		"func":  "addConnHandler",
		"event": cfg.EventWSConnAdded,
	}).Infof("addr: %v", conn.RemoteAddr())
}

func (s *StreamService) listenPubs(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenPubs",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	go s.store.Subscribe(s.sub.Chan, s.sub.ChanName)

	for {
		select {
		case msg := <-s.sub.Chan:
			go s.stream(ctx, msg)

		case <-ctx.Done():
			return
		}
	}
}

func (s *StreamService) stream(ctx context.Context, msg []byte) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "stream",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	var dev DevData
	if err := json.Unmarshal(msg, &dev); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "stream",
		}).Errorf("Unmarshal() failed: %s", err)
		return err
	}

	if _, ok := s.conns.idConns[DevID(dev.Meta.MAC)]; ok {
		for _, conn := range s.conns.idConns[DevID(dev.Meta.MAC)].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.conns.idConns[DevID(dev.Meta.MAC)].Lock()
				err := conn.WriteMessage(1, msg)
				s.conns.idConns[DevID(dev.Meta.MAC)].Unlock()
				if err != nil {
					s.log.WithFields(logrus.Fields{
						"func":  "stream",
						"event": cfg.EventWSConnRemoved,
					}).Infof("addr: %v", conn.RemoteAddr())
					s.conns.closedConns <- conn
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
			s.log.WithFields(logrus.Fields{
				"func":  "listenClosedConns",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	for {
		select {
		case conn := <-s.conns.closedConns:
			for mac, connList := range s.conns.idConns {
				if ok := connList.removeConn(conn); ok {
					s.conns.checkIDConns(mac)
					break
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

type streamConns struct {
	sync.RWMutex
	closedConns chan *websocket.Conn
	idConns     map[DevID]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		closedConns: make(chan *websocket.Conn),
		idConns:     make(map[DevID]*connList),
	}
}

func (c *streamConns) checkIDConns(id DevID) {
	c.Lock()
	if len(c.idConns[id].Conns) == 0 {
		delete(c.idConns, id)
	}
	c.Unlock()
}

type connList struct {
	sync.RWMutex
	Conns []*websocket.Conn
}

func (l *connList) addConn(c *websocket.Conn) {
	l.Lock()
	l.Conns = append(l.Conns, c)
	l.Unlock()
}

func (l *connList) removeConn(c *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	for i, c := range l.Conns {
		if c == c {
			l.Conns = append(l.Conns[:i], l.Conns[i+1:]...)
			return true
		}
	}
	return false
}
