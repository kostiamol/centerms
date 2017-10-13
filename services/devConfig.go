package services

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"os"

	"golang.org/x/net/context"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/pb"
	"google.golang.org/grpc"
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

type DevConfigService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
	ConnPool   ConnPool
	Messages   chan []string
}

func NewDevConfigServer(s entities.Server, ds entities.DevStorage, c entities.ServicesController, l *logrus.Logger,
	r *time.Ticker, msgs chan []string) *DevConfigService {
	l.Out = os.Stdout

	return &DevConfigService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
		Messages:   msgs,
	}
}

func (s *DevConfigService) Run() {
	s.Log.Infof("DevConfigService has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DevConfigService: Run(): panic(): %s", r)
			cancel()
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	s.ConnPool.Init()
	go s.listenConfig(ctx, entities.DevConfigChan, s.Messages)

	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		s.Log.Errorf("DevDataService: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				s.Log.Errorf("DevDataService: Run(): Listen() has failed: %s", err)
			}
		}
		s.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, s)
	gs.Serve(ln)
}

func (s *DevConfigService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *DevConfigService) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DevConfigService is down")
	s.Controller.Terminate()
}

func (s *DevConfigService) SetDevInitConfig(ctx context.Context, r *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevConfigService: sendInitConfig(): storage connection hasn't been established: %s", err)
		return nil, err
	}
	defer conn.CloseConn()

	req := entities.DevMeta{
		Type: r.Meta.Type,
		Name: r.Meta.Name,
		MAC:  r.Meta.Mac,
	}

	var dc *entities.DevConfig
	if ok, err := conn.DevIsRegistered(&req); ok {
		if err != nil {
			s.Log.Errorf("DevConfigService: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return nil, err
		}

		dc, err = conn.GetDevConfig(&req)
		if err != nil {
			s.Log.Errorf("DevConfigService: sendInitConfig(): GetDevConfig() has failed: %s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.Log.Errorf("DevConfigService: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return nil, err
		}

		dc, err = conn.GetDevDefaultConfig(&req)
		if err != nil {
			s.Log.Errorf("DevConfigService: sendInitConfig(): GetDevDefaultConfig() has failed: %s", err)
			return nil, err
		}

		s.Log.Printf("new device: Meta: %+v, DevConfig: %+v", req, dc)
		if err = conn.SetDevConfig(&req, dc); err != nil {
			s.Log.Errorf("DevConfigService: sendInitConfig(): SetDevConfig() has failed: %s", err)
			return nil, err
		}
	}

	return &pb.SetDevInitConfigResponse{
		Config: dc.Data,
	}, nil
}

func (s *DevConfigService) SaveDevData(ctx context.Context, r *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	return nil, nil
}

func (s *DevConfigService) listenConfig(ctx context.Context, channel string, msg chan []string) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevConfigService: listenConfig(): storage connection hasn't been established: ", err)
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
					s.Log.Errorf("DevConfigService: listenConfig(): DevConfig unmarshalling has failed: ", err)
					return
				}
				go s.sendConfigPatch(&dc)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevConfigService) sendConfigPatch(c *entities.DevConfig) {
	conn := s.ConnPool.GetConn(c.MAC)
	if conn == nil {
		s.Log.Errorf("DevConfigService: sendConfigPatch(): there isn't device connection with MAC [%s] in the pool", c.MAC)
		return
	}

	if _, err := conn.Write(c.Data); err != nil {
		s.Log.Errorf("DevConfigService: sendConfigPatch(): DevConfig.Data writing has failed: %s", err)
		s.ConnPool.RemoveConn(c.MAC)
		return
	}
	s.Log.Infof("send config patch: %s for device with MAC %s", c.Data, c.MAC)
}
