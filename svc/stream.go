package svc

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/cfg"

	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"context"

	"fmt"
)

type (
	// CfgSubscriber is a contract for the configuration subscriber.
	CfgSubscriber interface {
		Subscribe(c chan []byte, channel ...string) error
	}

	// StreamServiceCfg is used to initialize an instance of streamService.
	StreamServiceCfg struct {
		Log        log.Logger
		Ctrl       Ctrl
		Subscriber CfgSubscriber
		SubChan    string
		PortWS     uint64
	}

	// streamService is used to deal with streaming of data from the device to web client (dashboard).
	streamService struct {
		log        log.Logger
		ctrl       Ctrl
		subscriber CfgSubscriber
		sub        subscription
		portWS     uint64
		conns      streamConns
		upgrader   websocket.Upgrader
	}

	devID string
)

// NewStreamService creates and initializes a new instance of streamService service.
func NewStreamService(c *StreamServiceCfg) *streamService { // nolint
	return &streamService{
		portWS:     c.PortWS,
		subscriber: c.Subscriber,
		ctrl:       c.Ctrl,
		log:        c.Log.With("component", "stream"),
		conns:      *newStreamConns(),
		sub: subscription{
			ChanName: c.SubChan,
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
func (s *streamService) Run() {
	s.log.With("event", cfg.EventComponentStarted).
		Infof("is running on websocket port [%d]", s.portWS)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("%s", r)
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
		Handler: r,
		Addr:    fmt.Sprintf(":%d", s.portWS),
	}
	s.log.Fatal(srv.ListenAndServe())
}

func (s *streamService) listenTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *streamService) terminate() {
	s.log.With("event", cfg.EventComponentShutdown).Info("is down")
	_ = s.log.Flush()
	s.ctrl.Terminate()
}

func (s *streamService) addConnHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.Split(r.URL.String(), "/")
	if len(uri) < 3 {
		s.log.Errorf("url isn't complete")
		return
	}

	id := devID(uri[2])

	s.conns.Lock()
	if _, ok := s.conns.idConns[id]; !ok {
		s.conns.idConns[id] = new(connList)
	}
	s.conns.Unlock()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Errorf("addConnHandler(): Upgrade() failed %s", err)
		return
	}
	s.conns.idConns[id].addConn(conn)
	s.log.With("event", cfg.EventWSConnAdded).Infof("addConnHandler(): addr: %v", conn.RemoteAddr())
}

func (s *streamService) listenPubs(ctx context.Context) {
	go s.subscriber.Subscribe(s.sub.Chan, s.sub.ChanName) // nolint

	for {
		select {
		case msg := <-s.sub.Chan:
			go s.stream(ctx, msg)
		case <-ctx.Done():
			return
		}
	}
}

func (s *streamService) stream(ctx context.Context, msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("stream(): %s", r)
			s.terminate()
		}
	}()

	var d DevData
	if err := json.Unmarshal(msg, &d); err != nil {
		s.log.Errorf("stream(): Unmarshal() failed: %s", err)
		return
	}

	if _, ok := s.conns.idConns[devID(d.Meta.MAC)]; ok {
		for _, conn := range s.conns.idConns[devID(d.Meta.MAC)].Conns {
			select {
			case <-ctx.Done():
				return
			default:
				s.conns.idConns[devID(d.Meta.MAC)].Lock()
				err := conn.WriteMessage(1, msg)
				s.conns.idConns[devID(d.Meta.MAC)].Unlock()
				if err != nil {
					s.log.With("event", cfg.EventWSConnRemoved).
						Infof("stream(): addr: %v", conn.RemoteAddr())
					s.conns.closedConns <- conn
					return
				}
			}
		}
		return
	}
	return
}

func (s *streamService) listenClosedConns(ctx context.Context) {
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
	idConns     map[devID]*connList
}

func newStreamConns() *streamConns {
	return &streamConns{
		closedConns: make(chan *websocket.Conn),
		idConns:     make(map[devID]*connList),
	}
}

func (c *streamConns) checkIDConns(id devID) {
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

func (l *connList) removeConn(conn *websocket.Conn) bool {
	l.Lock()
	defer l.Unlock()
	for i, c := range l.Conns {
		if conn == c {
			l.Conns = append(l.Conns[:i], l.Conns[i+1:]...)
			return true
		}
	}
	return false
}
