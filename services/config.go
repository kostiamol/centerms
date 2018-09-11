// Package services provides definitions for services that run on the center.
package services

import (
	"golang.org/x/net/context"

	"math/rand"
	"time"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	aggregate = "config_service"
	event     = "config_patched"
)

// ConfigService is used to deal with device configs.
type ConfigService struct {
	addr      entities.Address
	storage   entities.Storager
	ctrl      entities.ServiceController
	log       *logrus.Entry
	retry     time.Duration
	sub       entities.Subscription
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewConfigService creates and initializes a new instance of ConfigService.
func NewConfigService(addr entities.Address, st entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, retry time.Duration, subj string, agentName string, ttl time.Duration) *ConfigService {

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
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination and config patches.
func (s *ConfigService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

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

	s.runConsulAgent()
}

// GetAddr returns address of the service.
func (s *ConfigService) GetAddr() entities.Address {
	return s.addr
}

func (s *ConfigService) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	consulAgent := &consul.AgentServiceRegistration{
		Name: s.agentName,
		Port: s.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}
	s.agent = c.Agent()
	if err := s.agent.ServiceRegister(consulAgent); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": entities.EventPanic,
		}).Errorf("%s", err)
		panic("Consul init error")
	}
	go s.updateTTL(s.check)
}

func (s *ConfigService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *ConfigService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *ConfigService) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := s.agent.UpdateTTL("service:"+s.agentName, "", health); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Error(err)
	}
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

// SetDevInitConfig check's whether device is already registered in the system. If it's already registered,
// the func returns actual config. Otherwise it returns default config for that type of device.
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

	event := api.EventStore{
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
