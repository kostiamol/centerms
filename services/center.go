package services

import (
	"time"

	"os"

	"github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/api/grpcsvc"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/storages/redis"
	"github.com/pkg/errors"
)

const (
	centerAgentName = "center"
	redisAgentName  = "redis"
)

type CenterService struct {
	configService *ConfigService
	dataService   *DataService
	webService    *WebService
	streamService *StreamService
	ctrl          entities.ServiceController
	log           *logrus.Entry
	retry         time.Duration
	name          string
	ttl           time.Duration
	agent         *consul.Agent
}

func NewCenterService(ctrl entities.ServiceController, log *logrus.Logger, retry time.Duration, name string,
	ttl time.Duration) *CenterService {

	storage := storages.NewRedisStorage(storageServer, logrus.NewEntry(log), retry, redisAgentName, ttl)
	if err := storage.Init(); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "NewCenterService",
			"event": entities.EventStorageInit,
		}).Errorf("%s", err)
		os.Exit(1)
	}

	config := NewConfigService(
		entities.Address{
			Host: localhost,
			Port: *devConfigPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		retry,
		entities.DevConfigChan,
	)
	data := NewDataService(
		entities.Address{
			Host: localhost,
			Port: *devDataPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataChan,
	)
	stream := NewStreamService(
		entities.Address{
			Host: webHost,
			Port: *streamPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataChan,
	)
	web := NewWebService(
		entities.Address{
			Host: webHost,
			Port: *webPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevConfigChan,
	)

	return &CenterService{
		configService: config,
		dataService:   data,
		webService:    web,
		streamService: stream,
		ctrl:          ctrl,
		log:           log.WithFields(logrus.Fields{"service": "center"}),
		retry:         retry,
		name:          name,
	}
}

func (s *CenterService) Run() {
	go s.configService.Run()
	go s.dataService.Run()
	go s.streamService.Run()
	go s.webService.Run()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: s.configService,
		DataService:   s.dataService,
		Retry:         s.retry,
		Log:           s.log,
	})
}

func (s *CenterService) check() (bool, error) {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return false, errors.Errorf("Consul: %s", err)
	}
	agent := c.Agent()

	consulAgent := &consul.AgentServiceRegistration{
		Name: s.name,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}

	if err := agent.ServiceRegister(consulAgent); err != nil {
		return false, errors.Errorf("Consul: %s", err)
	}
	go s.updateTTL(check)

	return nil
}

func check() (bool, error) {
	// if
	return true, nil
}

func (s *CenterService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *CenterService) update(check func() (bool, error)) {
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

	if err := agent.UpdateTTL("service:"+s.name, "", health); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": entities.EventUpdConsulStatus,
		}).Error(err)
	}
}
