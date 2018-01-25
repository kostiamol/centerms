package services

import (
	"encoding/json"

	"golang.org/x/net/context"

	"math/rand"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/kostiamol/centerms/api/pb"
	"github.com/kostiamol/centerms/entities"
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
	Log           *logrus.Entry
	Sub           entities.Subscription
	RetryInterval time.Duration
}

func NewConfigService(srv entities.Server, storage entities.Storage, ctrl entities.ServiceController,
	log *logrus.Entry, retry time.Duration, subj string) *ConfigService {

	return &ConfigService{
		Server:        srv,
		Storage:       storage,
		Ctrl:          ctrl,
		Log:           log.WithFields(logrus.Fields{"service": "config"}),
		RetryInterval: retry,
		Sub: entities.Subscription{
			Subject: subj,
			Channel: make(chan []string),
		},
	}
}

func (s *ConfigService) Run() {
	s.Log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": "panic",
			}).Errorf("%s", r)

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
			s.Log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": "panic",
			}).Errorf("%s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Infoln("ConfigService is down")
	s.Ctrl.Terminate()
}

func (s *ConfigService) SetDevInitConfig(meta *entities.DevMeta) (*entities.DevConfig, error) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	if err := conn.SetDevMeta(meta); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}

	var dc *entities.DevConfig
	if ok, err := conn.DevIsRegistered(meta); ok {
		if err != nil {
			s.Log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		dc, err = conn.GetDevConfig(meta.MAC)
		if err != nil {
			s.Log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.Log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		dc, err = conn.GetDevDefaultConfig(meta)
		if err != nil {
			s.Log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		if err = conn.SetDevConfig(meta.MAC, dc); err != nil {
			s.Log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}
		s.Log.WithFields(logrus.Fields{
			"func":  "SetDevInitConfig",
			"event": "device_registered",
		}).Infof("device's meta: %+v", meta)
	}
	return dc, err
}

func (s *ConfigService) listenConfigPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "listenConfigPatches",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "listenConfigPatches",
		}).Errorf("%s", err)
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
					s.Log.WithFields(logrus.Fields{
						"func": "listenConfigPatches",
					}).Errorf("%s", err)
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
			s.Log.WithFields(logrus.Fields{
				"func":  "publishNewConfigPatchEvent",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "publishNewConfigPatchEvent",
		}).Error("nats connectivity status: DISCONNECTED")
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
	s.Log.WithFields(logrus.Fields{
		"func":  "publishNewConfigPatchEvent",
		"event": event,
	}).Infof("publish config patch: %s for device with MAC [%s]", dc.Data, dc.MAC)
}
