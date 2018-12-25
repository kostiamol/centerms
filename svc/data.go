package svc

import (
	"encoding/json"

	"github.com/kostiamol/centerms/cfg"

	"github.com/Sirupsen/logrus"

	"time"

	consul "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

// DevMeta is used to store device metadata: it's type, name (model) and MAC address.
type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

// DevData is used to store time of the request, device's metadata and the data it transfers.
type DevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// DataService is used to deal with device data.
type DataService struct {
	addr      Addr
	store     Storer
	ctrl      Ctrl
	log       *logrus.Entry
	pubChan   string
	agent     *consul.Agent
	agentName string
	ttl       time.Duration
}

// NewDataService creates and initializes a new instance of DataService service.
func NewDataService(a Addr, s Storer, c Ctrl, l *logrus.Entry, pubChan string, agentName string,
	ttl time.Duration) *DataService {

	return &DataService{
		addr:      a,
		store:     s,
		ctrl:      c,
		log:       l.WithFields(logrus.Fields{"component": "svc", "name": "data"}),
		pubChan:   pubChan,
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutine that listens for the service termination.
func (s *DataService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
	s.runConsulAgent()
}

// GetAddr returns address of the service.
func (s *DataService) GetAddr() Addr {
	return s.addr
}

func (s *DataService) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	r := &consul.AgentServiceRegistration{
		Name: s.agentName,
		Port: s.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: s.ttl.String(),
		},
	}
	s.agent = c.Agent()
	if err := s.agent.ServiceRegister(r); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	go s.updateTTL(s.check)
}

func (s *DataService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *DataService) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(s.ttl / 2)
	for range t.C {
		s.update(check)
	}
}

func (s *DataService) update(check func() (bool, error)) {
	var health string
	if ok, err := check(); !ok {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := s.agent.UpdateTTL("svc:"+s.agentName, "", health); err != nil {
		s.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
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
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	if err := s.store.CloseConn(); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "terminate",
		}).Errorf("%s", err)
	}

	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": cfg.EventSVCShutdown,
	}).Infoln("svc is down")
	s.ctrl.Terminate()
}

// SaveDevData is used to save device data in the store.
func (s *DataService) SaveDevData(data *DevData) {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	go s.pubDevData(data)
}

func (s *DataService) pubDevData(data *DevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "pubDevData",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(data)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
