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

	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StreamService is used to deal with streaming data from the device to web client (dashboard).
type StreamService struct {
	addr      entities.Address
	storage   entities.Storager
	ctrl      entities.ServiceController
	log       *logrus.Entry
	sub       entities.Subscription
	conns     streamConns
	upgrader  websocket.Upgrader
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewStreamService creates and initializes a new instance of StreamService.
func NewStreamService(srv entities.Address, st entities.Storager, ctrl entities.ServiceController, log *logrus.Entry,
	pubChan string, agentName string, ttl time.Duration) *StreamService {
	upg := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Host == srv.Host+":"+fmt.Sprint(srv.Port) {
				return true
			}
			return true
		},
	}

	return &StreamService{
		addr:    srv,
		storage: st,
		ctrl:    ctrl,
		log:     log.WithFields(logrus.Fields{"service": "stream"}),
		conns:   *newStreamConns(),
		sub: entities.Subscription{
			ChanName: pubChan,
			Channel:  make(chan []byte),
		},
		upgrader:  upg,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination, new device data,
// closed web client connections and publishing new device data to web clients with open connections.
func (s *StreamService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
	go s.listenPublications(ctx)
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

func (s *StreamService) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	consulAgent := &consul.AgentServiceRegistration{
		Name: s.agentName,
		Port: s.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}
	s.agent = c.Agent()
	if err := s.agent.ServiceRegister(consulAgent); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	go s.updateTTL(s.check)
}

func (s *StreamService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *StreamService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *StreamService) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := s.agent.UpdateTTL("service:"+s.agentName, "", health); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Error(err)
	}
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
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entities.EventSVCShutdown,
	}).Info("service is down")
	s.ctrl.Terminate()
}

func (s *StreamService) addConnHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "addConnHandler",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	uri := strings.Split(r.URL.String(), "/")
	id := entities.DevID(uri[2])

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
		"event": entities.EventWSConnAdded,
	}).Infof("addr: %v", conn.RemoteAddr())
}

func (s *StreamService) listenPublications(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenPublications",
				"event": entities.EventPanic,
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
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	var data entities.DevData
	if err := json.Unmarshal(msg, &data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "stream",
		}).Errorf("%s", err)
		return err
	}

	if _, ok := s.conns.idConns[entities.DevID(data.Meta.MAC)]; ok {
		for _, conn := range s.conns.idConns[entities.DevID(data.Meta.MAC)].Conns {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.conns.idConns[entities.DevID(data.Meta.MAC)].Lock()
				err := conn.WriteMessage(1, msg)
				s.conns.idConns[entities.DevID(data.Meta.MAC)].Unlock()
				if err != nil {
					s.log.WithFields(logrus.Fields{
						"func":  "stream",
						"event": entities.EventWSConnRemoved,
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
				"event": entities.EventPanic,
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
	idConns     map[entities.DevID]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		closedConns: make(chan *websocket.Conn),
		idConns:     make(map[entities.DevID]*connList),
	}
}

func (c *streamConns) checkIDConns(id entities.DevID) {
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
