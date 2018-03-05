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

func NewConfigService(addr entities.Address, st entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, retry time.Duration, subj string) *ConfigService {

	return &ConfigService{
		addr:    addr,
		storage: st,
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
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

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
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entities.EventSVCShutdown,
	}).Infoln("service is down")
	s.ctrl.Terminate()
}

func (s *ConfigService) SetDevInitConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	if err := conn.SetDevMeta(m); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitConfig",
		}).Errorf("%s", err)
		return nil, err
	}

	var (
		config *entities.DevConfig
		id     = entities.DevID(m.MAC)
	)
	if ok, err := conn.DevIsRegistered(m); ok {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		config, err = conn.GetDevConfig(id)
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

		config, err = conn.GetDevDefaultConfig(m)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}

		if err = conn.SetDevConfig(id, config); err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitConfig",
			}).Errorf("%s", err)
			return nil, err
		}
		s.log.WithFields(logrus.Fields{
			"func":  "SetDevInitConfig",
			"event": entities.EventDevRegistered,
		}).Infof("device's meta: %+v", m)
	}
	return config, err
}

func (s *ConfigService) listenConfigPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenConfigPatches",
				"event": entities.EventPanic,
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

func (s *ConfigService) publishNewConfigPatchEvent(c *entities.DevConfig) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "publishNewConfigPatchEvent",
				"event": entities.EventPanic,
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
		AggregateId:   c.MAC,
		AggregateType: aggregate,
		EventId:       uuid.NewV4().String(),
		EventType:     event,
		EventData:     string(c.Data),
	}
	subject := "ConfigService.Patch." + c.MAC
	data, _ := proto.Marshal(&event)

	conn.Publish(subject, data)
	s.log.WithFields(logrus.Fields{
		"func":  "publishNewConfigPatchEvent",
		"event": entities.EventConfigPatchCreated,
	}).Infof("config patch [%s] for device with id [%s]", c.Data, c.MAC)
}
