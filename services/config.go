package services

import (
	"golang.org/x/net/context"

	"math/rand"
	"time"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/kostiamol/centerms/api/pb"
	"github.com/kostiamol/centerms/entities"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	aggregate = "config_service"
	event     = "config_patched"
)

type ConfigService struct {
	addr    entities.Address
	storage entities.Storager
	ctrl    entities.ServiceController
	log     *logrus.Entry
	sub     entities.Subscription
	retry   time.Duration
}

func NewConfigService(addr entities.Address, storage entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, retry time.Duration, subj string) *ConfigService {

	return &ConfigService{
		addr:    addr,
		storage: storage,
		ctrl:    ctrl,
		log:     log.WithFields(logrus.Fields{"service": "config"}),
		retry:   retry,
		sub: entities.Subscription{
			ChanName: subj,
			Channel:  make(chan []byte),
		},
	}
}

func (s *ConfigService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
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

func (s *ConfigService) GetAddr() entities.Address {
	return s.addr
}

func (s *ConfigService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *ConfigService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": "panic",
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Infoln("ConfigService is down")
	s.ctrl.Terminate()
}

func (s *ConfigService) SetDevInitConfig(meta *entities.DevMeta) (*entities.DevConfig, error) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	if err := conn.SetDevMeta(meta); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}

	var (
		dc *entities.DevConfig
		id = entities.DevID(meta.MAC)
	)
	if ok, err := conn.DevIsRegistered(meta); ok {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		dc, err = conn.GetDevConfig(id)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		dc, err = conn.GetDevDefaultConfig(meta)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		if err = conn.SetDevConfig(id, dc); err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}
		s.log.WithFields(logrus.Fields{
			"func":  "SetDevInitConfig",
			"event": "device_registered",
		}).Infof("device's meta: %+v", meta)
	}
	return dc, err
}

func (s *ConfigService) listenConfigPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenConfigPatches",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenConfigPatches",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	go conn.Subscribe(s.sub.Channel, s.sub.ChanName)

	var config entities.DevConfig
	for {
		select {
		case msg := <-s.sub.Channel:
			if err := json.Unmarshal(msg, &config); err != nil {
				s.log.WithFields(logrus.Fields{
					"func": "listenConfigPatches",
				}).Errorf("%s", err)
			} else {
				go s.publishNewConfigPatchEvent(&config)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ConfigService) publishNewConfigPatchEvent(dc *entities.DevConfig) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "publishNewConfigPatchEvent",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishNewConfigPatchEvent",
		}).Error("nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
	}
	defer conn.Close()

	event := pb.EventStore{
		AggregateId:   dc.MAC,
		AggregateType: aggregate,
		EventId:       uuid.NewV4().String(),
		EventType:     event,
		EventData:     string(dc.Data),
	}
	subject := "ConfigService.Patch." + dc.MAC
	data, _ := proto.Marshal(&event)

	conn.Publish(subject, data)
	s.log.WithFields(logrus.Fields{
		"func":  "publishNewConfigPatchEvent",
		"event": event,
	}).Infof("publish config patch: %s for device with MAC [%s]", dc.Data, dc.MAC)
}
