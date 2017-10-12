package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"os"

	"context"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"google.golang.org/grpc"
	"github.com/giperboloid/centerms/pb"
)

type ConnPool struct {
	sync.Mutex
	conn map[string]net.Conn
}

func (p *ConnPool) Init() {
	p.Lock()
	p.conn = make(map[string]net.Conn)
	p.Unlock()
}

func (p *ConnPool) AddConn(cn net.Conn, key string) {
	p.Lock()
	p.conn[key] = cn
	p.Unlock()
}

func (p *ConnPool) GetConn(key string) net.Conn {
	p.Lock()
	defer p.Unlock()
	return p.conn[key]
}

func (p *ConnPool) RemoveConn(key string) {
	p.Lock()
	delete(p.conn, key)
	p.Unlock()
}

type DevConfigServer struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServersController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
	ConnPool   ConnPool
	Messages   chan []string
}

func NewDevConfigServer(s entities.Server, ds entities.DevStorage, c entities.ServersController, l *logrus.Logger,
	r *time.Ticker, msgs chan []string) *DevConfigServer {
	l.Out = os.Stdout

	return &DevConfigServer{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
		Messages:   msgs,
	}
}

func (s *DevConfigServer) Run() {
	s.Log.Infof("DevConfigServer has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DevConfigServer: Run(): panic(): %s", r)
			cancel()
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	s.ConnPool.Init()
	go s.listenConfig(ctx, entities.DevConfigChan, s.Messages)

	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		s.Log.Errorf("DevDataServer: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				s.Log.Errorf("DevDataServer: Run(): Listen() has failed: %s", err)
			}
		}
		s.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterDevServiceServer(gs, s)
	gs.Serve(ln)
}

func (s *DevConfigServer) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *DevConfigServer) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DevConfigServer is down")
	s.Controller.Terminate()
}

func (s *DevConfigServer) sendInitConfig(c net.Conn) {
	var req entities.Request
	if err := json.NewDecoder(c).Decode(&req); err != nil {
		s.Log.Errorf("DevConfigServer: sendInitConfig(): Request marshalling has failed: %s", err)
		return
	}
	s.ConnPool.AddConn(c, req.Meta.MAC)

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevConfigServer: sendInitConfig(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	var dc *entities.DevConfig
	if ok, err := conn.DevIsRegistered(&req.Meta); ok {
		if err != nil {
			s.Log.Errorf("DevConfigServer: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return
		}

		dc, err = conn.GetDevConfig(&req.Meta)
		if err != nil {
			s.Log.Errorf("DevConfigServer: sendInitConfig(): GetDevConfig() has failed: %s", err)
			return
		}
	} else {
		if err != nil {
			s.Log.Errorf("DevConfigServer: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return
		}

		dc, err = conn.GetDevDefaultConfig(&req.Meta)
		if err != nil {
			s.Log.Errorf("DevConfigServer: sendInitConfig(): GetDevDefaultConfig() has failed: %s", err)
			return
		}

		s.Log.Printf("new device: Meta: %+v, DevConfig: %+v", req.Meta, dc)
		if err = conn.SetDevConfig(&req.Meta, dc); err != nil {
			s.Log.Errorf("DevConfigServer: sendInitConfig(): SetDevConfig() has failed: %s", err)
			return
		}
	}

	if _, err = c.Write(dc.Data); err != nil {
		s.Log.Errorf("DevConfigServer: sendInitConfig(): DevConfig.Data writing has failed: %s", err)
		return
	}
}

func (s *DevConfigServer) listenConfig(ctx context.Context, channel string, msg chan []string) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevConfigServer: listenConfig(): storage connection hasn't been established: ", err)
		return
	}
	defer conn.CloseConn()

	var dc entities.DevConfig
	conn.Subscribe(msg, channel)
	for {
		select {
		case msg := <-msg:
			if msg[0] == "message" {
				if err := json.Unmarshal([]byte(msg[2]), &dc); err != nil {
					s.Log.Errorf("DevConfigServer: listenConfig(): DevConfig unmarshalling has failed: ", err)
					return
				}
				go s.sendConfigPatch(&dc)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevConfigServer) sendConfigPatch(c *entities.DevConfig) {
	conn := s.ConnPool.GetConn(c.MAC)
	if conn == nil {
		s.Log.Errorf("DevConfigServer: sendConfigPatch(): there isn't device connection with MAC [%s] in the pool", c.MAC)
		return
	}

	if _, err := conn.Write(c.Data); err != nil {
		s.Log.Errorf("DevConfigServer: sendConfigPatch(): DevConfig.Data writing has failed: %s", err)
		s.ConnPool.RemoveConn(c.MAC)
		return
	}
	s.Log.Infof("send config patch: %s for device with MAC %s", c.Data, c.MAC)
}
