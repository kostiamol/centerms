package svc

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kostiamol/centerms/store/dev"

	"github.com/kostiamol/centerms/metric"

	"github.com/kostiamol/centerms/log"

	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"context"

	"fmt"
)

type (
	// StreamServiceCfg is used to initialize an instance of streamService.
	StreamServiceCfg struct {
		Log     log.Logger
		Ctrl    Ctrl
		Metric  *metric.Metric
		SubChan <-chan *dev.Data
		PortWS  uint32
	}

	// streamService is used to deal with streaming of data from the device to web client (dashboard).
	streamService struct {
		log      log.Logger
		ctrl     Ctrl
		metric   *metric.Metric
		subChan  <-chan *dev.Data
		portWS   uint32
		conns    *streamConns
		upgrader websocket.Upgrader
	}

	devID string
)

// NewStreamService creates and initializes a new instance of streamService service.
func NewStreamService(c *StreamServiceCfg) *streamService { // nolint
	return &streamService{
		log:     c.Log.With("component", "stream"),
		ctrl:    c.Ctrl,
		metric:  c.Metric,
		subChan: c.SubChan,
		portWS:  c.PortWS,
		conns:   newStreamConns(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Run launches the service by running goroutines for listening to the service termination, new device data,
// closed web client connections and publishing new device data to web clients with open connections.
func (s *streamService) Run() {
	s.log.With("event", log.EventComponentStarted).Infof("websocket port [%d]", s.portWS)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func Run: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
			cancel()
			s.ctrl.Terminate()
		}
	}()

	go s.listenToTermination()
	go s.listenToDataChanges(ctx)
	go s.listenToClosedConns(ctx)

	r := mux.NewRouter()
	r.HandleFunc("/devices/{id}", s.addConnHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", s.portWS),
	}

	if err := srv.ListenAndServe(); err != nil {
		s.log.Errorf("func ListenAndServe: %s", err)
		s.ctrl.Terminate()
	}
}

func (s *streamService) listenToTermination() {
	<-s.ctrl.StopChan
	s.log.With("event", log.EventComponentShutdown).Infof("")
	_ = s.log.Flush()
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
		s.log.Errorf("func addConnHandler: func Upgrade: %s", err)
		return
	}
	s.conns.idConns[id].addConn(conn)
	s.log.With("event", log.EventWSConnAdded, "addr", conn.RemoteAddr().String())
}

func (s *streamService) listenToDataChanges(ctx context.Context) {
	for {
		select {
		case msg := <-s.subChan:
			go s.stream(ctx, msg)
		case <-ctx.Done():
			return
		}
	}
}

func (s *streamService) stream(ctx context.Context, d *dev.Data) {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func stream: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
			s.ctrl.Terminate()
		}
	}()

	b, err := json.Marshal(d)
	if err != nil {
		s.log.Errorf("func Marshal: %s", err)
	}

	if _, ok := s.conns.idConns[devID(d.Meta.MAC)]; ok {
		for _, conn := range s.conns.idConns[devID(d.Meta.MAC)].Conns {
			select {
			case <-ctx.Done():
				return
			default:
				s.conns.idConns[devID(d.Meta.MAC)].Lock()
				err := conn.WriteMessage(1, b)
				s.conns.idConns[devID(d.Meta.MAC)].Unlock()
				if err != nil {
					s.log.With("event", log.EventWSConnRemoved, "addr", conn.RemoteAddr().String())
					s.conns.closedConns <- conn
					return
				}
			}
		}
		return
	}
}

func (s *streamService) listenToClosedConns(ctx context.Context) {
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
