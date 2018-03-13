package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
	"golang.org/x/net/context"
)

// DataService is used to deal with device data.
type DataService struct {
	addr      entities.Address
	storage   entities.Storager
	ctrl      entities.ServiceController
	log       *logrus.Entry
	pubChan   string
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewDataService creates and initializes a new instance of DataService.
func NewDataService(srv entities.Address, st entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, pubChan string, agentName string, ttl time.Duration) *DataService {

	return &DataService{
		addr:      srv,
		storage:   st,
		ctrl:      ctrl,
		log:       log.WithFields(logrus.Fields{"service": "data"}),
		pubChan:   pubChan,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutine that listens for the service termination.
func (s *DataService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

	_, cancel := context.WithCancel(context.Background())
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

	s.runConsulAgent()
}

// GetAddr returns address of the service.
func (s *DataService) GetAddr() entities.Address {
	return s.addr
}

func (s *DataService) runConsulAgent() {
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

func (s *DataService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *DataService) updateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.ttl / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *DataService) update(check func() (bool, error)) {
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

func (s *DataService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *DataService) terminate() {
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

// SaveDevData is used to save device data in the storage.
func (s *DataService) SaveDevData(d *entities.DevData) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(d); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}

	go s.publishDevData(d)
}

func (s *DataService) publishDevData(d *entities.DevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "publishDevData",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(d)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
