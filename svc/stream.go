package svc

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

	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entity"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Stream is used to deal with streaming data from the device to web client (dashboard).
type Stream struct {
	addr      entity.Addr
	store     entity.Storer
	ctrl      Ctrl
	log       *logrus.Entry
	sub       entity.Subscription
	conns     streamConns
	upgrader  websocket.Upgrader
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewStream creates and initializes a new instance of Stream service.
func NewStream(a entity.Addr, s entity.Storer, c Ctrl, l *logrus.Entry,
	pubChan string, agentName string, ttl time.Duration) *Stream {
	upg := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == a.Host+":"+fmt.Sprint(a.Port) {
				return true
			}
			return true
		},
	}

	return &Stream{
		addr:  a,
		store: s,
		ctrl:  c,
		log:   l.WithFields(logrus.Fields{"service": "stream"}),
		conns: *newStreamConns(),
		sub: entity.Subscription{
			ChanName: pubChan,
			Chan:     make(chan []byte),
		},
		upgrader:  upg,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination, new device data,
// closed web client connections and publishing new device data to web clients with open connections.
func (s *Stream) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entity.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
	go s.listenPubs(ctx)
	go s.listenClosedConns(ctx)

	s.runConsulAgent()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.addConnHandler)
	// for Prometheus
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      r,
		Addr:         s.addr.Host + ":" + fmt.Sprint(s.addr.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	s.log.Fatal(srv.ListenAndServe())
}

func (s *Stream) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entity.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	a := &consul.AgentServiceRegistration{
		Name: s.agentName,
		Port: s.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}
	s.agent = c.Agent()
	if err := s.agent.ServiceRegister(a); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entity.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	go s.updateTTL(s.check)
}

func (s *Stream) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *Stream) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(s.ttl / 2)
	for range t.C {
		s.update(check)
	}
}

func (s *Stream) update(check func() (bool, error)) {
	var h string
	ok, err := check()
	if !ok {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entity.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		h = consul.HealthCritical
	} else {
		h = consul.HealthPassing
	}

	if err := s.agent.UpdateTTL("svc:"+s.agentName, "", h); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entity.EventUpdConsulStatus,
		}).Error(err)
	}
}

func (s *Stream) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *Stream) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	s.store.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entity.EventSVCShutdown,
	}).Info("svc is down")
	s.ctrl.Terminate()
}

func (s *Stream) addConnHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "addConnHandler",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	uri := strings.Split(r.URL.String(), "/")
	id := entity.DevID(uri[2])

	s.conns.Lock()
	if _, ok := s.conns.idConns[id]; !ok {
		s.conns.idConns[id] = new(connList)
	}
	s.conns.Unlock()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "addConnHandler",
		}).Errorf("%s", err)
		return
	}
	s.conns.idConns[id].addConn(conn)
	s.log.WithFields(logrus.Fields{
		"func":  "addConnHandler",
		"event": entity.EventWSConnAdded,
	}).Infof("addr: %v", conn.RemoteAddr())
}

func (s *Stream) listenPubs(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenPubs",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenPubs",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()
	go conn.Subscribe(s.sub.Chan, s.sub.ChanName)

	for {
		select {
		case msg := <-s.sub.Chan:
			go s.stream(ctx, msg)

		case <-ctx.Done():
			return
		}
	}
}

func (s *Stream) stream(ctx context.Context, msg []byte) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "stream",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	var data entity.DevData
	if err := json.Unmarshal(msg, &data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "stream",
		}).Errorf("%s", err)
		return err
	}

	if _, ok := s.conns.idConns[entity.DevID(data.Meta.MAC)]; ok {
		for _, conn := range s.conns.idConns[entity.DevID(data.Meta.MAC)].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.conns.idConns[entity.DevID(data.Meta.MAC)].Lock()
				err := conn.WriteMessage(1, msg)
				s.conns.idConns[entity.DevID(data.Meta.MAC)].Unlock()
				if err != nil {
					s.log.WithFields(logrus.Fields{
						"func":  "stream",
						"event": entity.EventWSConnRemoved,
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

func (s *Stream) listenClosedConns(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenClosedConns",
				"event": entity.EventPanic,
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
	idConns     map[entity.DevID]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		closedConns: make(chan *websocket.Conn),
		idConns:     make(map[entity.DevID]*connList),
	}
}

func (c *streamConns) checkIDConns(id entity.DevID) {
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
