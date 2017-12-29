package services

import (
	"encoding/json"

	"os"

	"golang.org/x/net/context"

	"math/rand"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/pb"
	"github.com/kostiamol/centerms/entities"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	aggregate = "Config"
	event     = "ConfigPatched"
)

type ConfigService struct {
	Server        entities.Server
	Storage       entities.Storage
	Ctrl          entities.ServiceController
	Log           *logrus.Logger
	Sub           entities.Subscription
	RetryInterval time.Duration
}

func NewConfigService(srv entities.Server, st entities.Storage, c entities.ServiceController,
	l *logrus.Logger, retry time.Duration, subject string) *ConfigService {

	l.Out = os.Stdout
	return &ConfigService{
		Server:        srv,
		Storage:       st,
		Ctrl:          c,
		Log:           l,
		RetryInterval: retry,
		Sub: entities.Subscription{
			Subject: subject,
			Channel: make(chan []string),
		},
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
	go s.listenConfigPatches(ctx)
}

func (s *ConfigService) listenTermination() {
	for {
		select {
		case <-s.Ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *ConfigService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: terminate(): panic(): %s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.Infoln("ConfigService is down")
	s.Ctrl.Terminate()
}

func (s *ConfigService) SetDevInitConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("ConfigService: sendInitConfig(): storage connection hasn't been established: %s", err)
		return nil, err
	}
	defer conn.CloseConn()

	if err := conn.SetDevMeta(m); err != nil {
		return nil, err
	}

	var dc *entities.DevConfig
	if ok, err := conn.DevIsRegistered(m); ok {
		if err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): DevIsRegistered() has failed: %s", err)
			return nil, err
		}

		dc, err = conn.GetDevConfig(m.MAC)
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

		if err = conn.SetDevConfig(m.MAC, dc); err != nil {
			s.Log.Errorf("ConfigService: sendInitConfig(): SetDevConfig() has failed: %s", err)
			return nil, err
		}
		s.Log.Infof("new device is registered: %+v", m)
	}
	return dc, err
}

func (s *ConfigService) listenConfigPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: listenConfigPatches(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("ConfigService: listenConfigPatches(): storage connection hasn't been established: ", err)
		return
	}
	defer conn.CloseConn()
	conn.Subscribe(s.Sub.Channel, s.Sub.Subject)

	var dc entities.DevConfig
	for {
		select {
		case msg := <-s.Sub.Channel:
			if msg[0] == "message" {
				if err := json.Unmarshal([]byte(msg[2]), &dc); err != nil {
					s.Log.Errorf("ConfigService: listenConfigPatches(): DevConfig unmarshalling has failed: ", err)
					return
				}
				go s.publishNewConfigPatchEvent(&dc)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ConfigService) publishNewConfigPatchEvent(dc *entities.DevConfig) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: publishNewConfigPatchEvent(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.Log.Error("ConfigService: publishNewConfigPatchEvent(): nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
	}

	defer conn.Close()

	event := api.EventStore{
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
