package svc

import (
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/cfg"
)

// MeshAgent represents a service mesh agent.
type MeshAgent struct {
	name  string
	port  int
	agent *consul.Agent
	ttl   time.Duration
	log   *logrus.Entry
}

// NewAgent creates and initializes a new instance of MeshAgent.
func NewAgent(name string, port int, ttl time.Duration, log *logrus.Entry) *MeshAgent {
	return &MeshAgent{
		name: name,
		port: port,
		ttl:  ttl,
		log:  log.WithFields(logrus.Fields{"component": "svc", "name": "meshAgent"}),
	}
}

// Run launches the agent.
func (a *MeshAgent) Run() {
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("NewClient() failed: %s", err)
	}
	agentReg := &consul.AgentServiceRegistration{
		Name: a.name,
		Port: a.port,
		Check: &consul.AgentServiceCheck{
			TTL: a.ttl.String(),
		},
	}
	a.agent = client.Agent()
	if err := a.agent.ServiceRegister(agentReg); err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "Run",
			"event": cfg.EventPanic,
		}).Errorf("ServiceRegister() failed: %s", err)
	}
	go a.updateTTL(check)
}

func check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (a *MeshAgent) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(a.ttl / 2)
	for range t.C {
		a.update(check)
	}
}

func (a *MeshAgent) update(check func() (bool, error)) {
	var health string
	if ok, err := check(); !ok {
		a.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Errorf("check() failed: %s", err)
		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := a.agent.UpdateTTL(a.name, "", health); err != nil {
		a.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Errorf("UpdateTTL() failed: %s", err)
	}
}
