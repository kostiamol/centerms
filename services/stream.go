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

	"github.com/kostiamol/centerms/entities"
)

type StreamService struct {
	addr     entities.Address
	storage  entities.Storager
	ctrl     entities.ServiceController
	log      *logrus.Entry
	sub      entities.Subscription
	conns    streamConns
	upgrader websocket.Upgrader
}

func NewStreamService(srv entities.Address, storage entities.Storager, ctrl entities.ServiceController, log *logrus.Entry,
	pubChan string) *StreamService {
	upg := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == srv.Host+":"+srv.Port {
				return true
			}
			return true
		},
	}

	return &StreamService{
		addr:    srv,
		storage: storage,
		ctrl:    ctrl,
		log:     log.WithFields(logrus.Fields{"service": "stream"}),
		conns:   *newStreamConns(),
		sub: entities.Subscription{
			ChanName: pubChan,
			Channel:  make(chan []byte),
		},
		upgrader: upg,
	}
}

func (s *StreamService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": "panic",
			}).Errorf("%s", r)
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
		Addr:         s.addr.Host + ":" + s.addr.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	s.log.Fatal(srv.ListenAndServe())
}

func (s *StreamService) GetAddr() entities.Address {
	return s.addr
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
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Info("StreamService is down")
	s.ctrl.Terminate()
}

func (s *StreamService) addConnHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "addConnHandler",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	uri := strings.Split(r.URL.String(), "/")
	mac := uri[2]

	s.conns.Lock()
	if _, ok := s.conns.macConns[mac]; !ok {
		s.conns.macConns[mac] = new(connList)
	}
	s.conns.Unlock()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "addConnHandler",
		}).Errorf("%s", err)
		return
	}
	s.conns.macConns[mac].addConn(conn)
	s.log.WithFields(logrus.Fields{
		"func":  "addConnHandler",
		"event": "ws_conn_added",
	}).Infof("addr: %v", conn.RemoteAddr())
}

func (s *StreamService) listenPublications(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenPublications",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenPublications",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()
	go conn.Subscribe(s.sub.Channel, s.sub.ChanName)

	for {
		select {
		case msg := <-s.sub.Channel:
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
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	var data entities.RawDevData
	if err := json.Unmarshal(msg, &data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "stream",
		}).Errorf("%s", err)
		return err
	}

	if _, ok := s.conns.macConns[data.Meta.MAC]; ok {
		for _, conn := range s.conns.macConns[data.Meta.MAC].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.conns.macConns[data.Meta.MAC].Lock()
				err := conn.WriteMessage(1, msg)
				s.conns.macConns[data.Meta.MAC].Unlock()
				if err != nil {
					s.log.WithFields(logrus.Fields{
						"func":  "stream",
						"event": "ws_conn_closed",
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
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	for {
		select {
		case conn := <-s.conns.closedConns:
			for mac, connList := range s.conns.macConns {
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

type streamConns struct {
	sync.RWMutex
	closedConns chan *websocket.Conn
	macConns    map[string]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		closedConns: make(chan *websocket.Conn),
		macConns:    make(map[string]*connList),
	}
}

func (c *streamConns) checkMACConns(mac string) {
	c.Lock()
	if len(c.macConns[mac].Conns) == 0 {
		delete(c.macConns, mac)
	}
	c.Unlock()
}

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
