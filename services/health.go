package services

import (
	"time"

	"github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
)

type HealthService struct {
	ctrl      entities.ServiceController
	log       *logrus.Entry
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

func NewHealthService(ctrl entities.ServiceController, log *logrus.Entry, agentName string,
	ttl time.Duration) *HealthService {

	return &HealthService{
		ctrl:      ctrl,
		log:       log.WithFields(logrus.Fields{"service": "health"}),
		agentName: agentName,
		ttl:       ttl,
	}
}

func (s *HealthService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("")

	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

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

func (s *HealthService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *HealthService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entities.EventSVCShutdown,
	}).Infoln("service is down")
	s.ctrl.Terminate()
}

func (s *HealthService) check() (bool, error) {
	// while all the services are alive - everything is ok
	return true, nil
}

func (s *HealthService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *HealthService) update(check func() (bool, error)) {
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
