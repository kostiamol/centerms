package services

import (
	"encoding/json"
	"net"

	"os"

	"golang.org/x/net/context"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/pb"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	aggregate = "Config"
	event     = "ConfigPatched"
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

func (p *ConnPool) AddConn(conn net.Conn, key string) {
	p.Lock()
	p.conn[key] = conn
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
	ConnPool   ConnPool
	Messages   chan []string
}

func NewConfigService(s entities.Server, ds entities.DevStorage, c entities.ServicesController,
	l *logrus.Logger) *ConfigService {

	l.Out = os.Stdout
	return &ConfigService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Messages:   make(chan []string),
	}
}

func (s *ConfigService) Run() {
	s.Log.Infof("ConfigService is running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: Run(): panic(): %s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()

	s.ConnPool.Init()
	go s.listenConfigPatch(ctx, entities.DevConfigChan, s.Messages)
}

func (s *ConfigService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *ConfigService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: terminate(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

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

		s.Log.Infof("new device is registered: %+v", m)
		if err = conn.SetDevConfig(m, dc); err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): SetDevConfig() has failed: %s", err)
			return nil, err
		}
	}

	return dc, err
}

func (s *ConfigService) listenConfigPatch(ctx context.Context, channel string, msg chan []string) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: listenConfigPatch(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("ConfigService: listenConfigPatch(): storage connection hasn't been established: ", err)
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
					s.Log.Errorf("ConfigService: listenConfigPatch(): DevConfig unmarshalling has failed: ", err)
					return
				}
				go s.publishConfigPatch(&dc)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ConfigService) publishConfigPatch(dc *entities.DevConfig) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: publishConfigPatch(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, _ := nats.Connect(nats.DefaultURL)
	defer conn.Close()

	event := pb.EventStore{
		AggregateId:   dc.MAC,
		AggregateType: aggregate,
		EventId:       uuid.NewV4().String(),
		EventType:     event,
		EventData:     string(dc.Data),
	}
	subject := "Config.Patch." + dc.MAC
	data, _ := proto.Marshal(&event)

	conn.Publish(subject, data)
	s.Log.Infof("publish config patch: %s for device with MAC [%s]", dc.Data, dc.MAC)
}
