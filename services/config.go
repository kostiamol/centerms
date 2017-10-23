package services

import (
	"encoding/json"
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

type ConfigService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
	ConnPool   ConnPool
	Messages   chan []string
}

func NewConfigService(s entities.Server, ds entities.DevStorage, c entities.ServicesController, l *logrus.Logger,
	r *time.Ticker) *ConfigService {
	l.Out = os.Stdout
	return &ConfigService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
		Messages:   make(chan []string),
	}
}

func (s *ConfigService) Run() {
	s.Log.Infof("ConfigService has started on host: [%s], port: [%d]", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: Run(): panic(): %s", r)
			cancel()
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	s.ConnPool.Init()
	go s.listenConfig(ctx, entities.DevConfigChan, s.Messages)
}

func (s *ConfigService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *ConfigService) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("ConfigService is down")
	s.Controller.Terminate()
}

func (s *ConfigService) SetDevInitConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("ConfigService: sendInitConfig(): storage connection hasn't been established: %s", err)
		return nil, err
	}
	defer conn.CloseConn()

	var dc *entities.DevConfig
	if ok, err := conn.DevIsRegistered(m); ok {
		if err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return nil, err
		}

		dc, err = conn.GetDevConfig(m)
		if err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): GetDevConfig() has failed: %s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return nil, err
		}

		dc, err = conn.GetDevDefaultConfig(m)
		if err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): GetDevDefaultConfig() has failed: %s", err)
			return nil, err
		}

		s.Log.Printf("new device: %+v", m)
		if err = conn.SetDevConfig(m, dc); err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): SetDevConfig() has failed: %s", err)
			return nil, err
		}
	}

	return dc, err
}

func (s *ConfigService) listenConfig(ctx context.Context, channel string, msg chan []string) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("ConfigService: listenConfig(): storage connection hasn't been established: ", err)
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
					s.Log.Errorf("ConfigService: listenConfig(): DevConfig unmarshalling has failed: ", err)
					return
				}
				go s.sendConfigPatch(&dc)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ConfigService) etalon(c *entities.DevConfig) {
	conn := s.ConnPool.GetConn(c.MAC)
	if conn == nil {
		s.Log.Errorf("ConfigService: sendConfigPatch(): there isn't device connection with MAC [%s] in the pool", c.MAC)
		return
	}

	if _, err := conn.Write(c.Data); err != nil {
		s.Log.Errorf("ConfigService: sendConfigPatch(): DevConfig.Data writing has failed: %s", err)
		s.ConnPool.RemoveConn(c.MAC)
		return
	}
	s.Log.Infof("send config patch: %s for device with MAC %s", c.Data, c.MAC)
}

func (s *ConfigService) sendConfigPatch(c *entities.DevConfig) {
	pbcp := pb.PatchDevConfigRequest{
		Config: c.Data,
	}
	conn := s.dialDevice()
	defer conn.Close()

	client := pb.NewDevServiceClient(conn)

	r, err := client.PatchDevConfig(context.Background(), &pbcp)
	if err != nil {
		s.Log.Errorf("ConfigService: sendConfigPatch(): PatchDevConfig() has failed: %s", err)
	}

	s.Log.Infof("device has received patched config with status: %s", r.Status)
}

func (s *ConfigService) dialDevice() *grpc.ClientConn {
	var count int
	conn, err := grpc.Dial(s.Server.Host+":"+"4040", grpc.WithInsecure())
	for err != nil {
		if count >= 5 {
			panic("ConfigService: dialDevice(): can't connect to the devicems")
		}
		time.Sleep(time.Second)
		conn, err = grpc.Dial("127.0.0.1"+":"+"4040", grpc.WithInsecure())
		if err != nil {
			s.Log.Errorf("getDial(): %s", err)
		}
		count++
		s.Log.Infof("reconnect count: %d", count)
	}
	return conn
}
