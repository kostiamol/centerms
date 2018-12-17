package svc

import (
	"encoding/json"

	"github.com/kostiamol/centerms/params"

	"github.com/kostiamol/centerms/api"

	"github.com/Sirupsen/logrus"

	"time"

	consul "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

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
func (d *DataService) Run() {
	d.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": params.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", d.addr.Host, d.addr.Port)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			d.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": params.EventPanic,
			}).Errorf("%s", r)
			cancel()
			d.terminate()
		}
	}()

	go d.listenTermination()
	d.runConsulAgent()
}

// GetAddr returns address of the service.
func (d *DataService) GetAddr() Addr {
	return d.addr
}

func (d *DataService) runConsulAgent() {
	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		d.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": params.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	agent := &consul.AgentServiceRegistration{
		Name: d.agentName,
		Port: d.addr.Port,
		Check: &consul.AgentServiceCheck{
			TTL: d.ttl.String(),
		},
	}
	d.agent = c.Agent()
	if err := d.agent.ServiceRegister(agent); err != nil {
		d.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": params.EventPanic,
		}).Errorf("%s", err)
		panic("consul init error")
	}
	go d.updateTTL(d.check)
}

func (d *DataService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (d *DataService) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(d.ttl / 2)
	for range t.C {
		d.update(check)
	}
}

func (d *DataService) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		d.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": params.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := d.agent.UpdateTTL("svc:"+d.agentName, "", health); err != nil {
		d.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": params.EventUpdConsulStatus,
		}).Error(err)
	}
}

func (d *DataService) listenTermination() {
	for {
		select {
		case <-d.ctrl.StopChan:
			d.terminate()
			return
		}
	}
}

func (d *DataService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			d.log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": params.EventPanic,
			}).Errorf("%s", r)
			d.ctrl.Terminate()
		}
	}()

	if err := d.store.CloseConn(); err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "terminate",
		}).Errorf("%s", err)
	}

	d.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": params.EventSVCShutdown,
	}).Infoln("svc is down")
	d.ctrl.Terminate()
}

// SaveDevData is used to save device data in the store.
func (d *DataService) SaveDevData(data *api.DevData) {
	conn, err := d.store.CreateConn()
	if err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(data); err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	go d.pubDevData(data)
}

func (d *DataService) pubDevData(data *api.DevData) error {
	defer func() {
		if r := recover(); r != nil {
			d.log.WithFields(logrus.Fields{
				"func":  "pubDevData",
				"event": params.EventPanic,
			}).Errorf("%s", r)
			d.terminate()
		}
	}()

	conn, err := d.store.CreateConn()
	if err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(data)
	if err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	if _, err = conn.Publish(b, d.pubChan); err != nil {
		d.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
