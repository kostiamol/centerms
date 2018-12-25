// Package svc provides definitions for services that run on the center.
package svc

import (
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/proto"
	"golang.org/x/net/context"

	"math/rand"
	"time"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	gproto "github.com/golang/protobuf/proto"
	consul "github.com/hashicorp/consul/api"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	aggregate = "cfg_svc"
	event     = "cfg_patched"
)

// DevCfg holds device's MAC address and config.
type DevCfg struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

// CfgService is used to deal with device configurations.
type CfgService struct {
	addr      Addr
	store     Storer
	ctrl      Ctrl
	log       *logrus.Entry
	retry     time.Duration
	sub       subscription
	agent     *consul.Agent
	agentName string
	ttl       time.Duration
}

// NewCfgService creates and initializes a new instance of CfgService service.
func NewCfgService(a Addr, s Storer, c Ctrl, l *logrus.Entry, retry time.Duration, subj string,
	agentName string, ttl time.Duration) *CfgService {

	return &CfgService{
		addr:  a,
		store: s,
		ctrl:  c,
		log:   l.WithFields(logrus.Fields{"component": "svc", "name": "cfg"}),
		retry: retry,
		sub: subscription{
			ChanName: subj,
			Chan:     make(chan []byte),
		},
		agentName: agentName,
		ttl:       ttl,
	}
}

// Run launches the service by running goroutines for listening the service termination and config patches.
func (s *CfgService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%d]", s.addr.Host, s.addr.Port)

	ctx, cancel := context.WithCancel(context.Background())
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
	go s.listenCfgPatches(ctx)
	s.runConsulAgent()
}

func (s *CfgService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *CfgService) terminate() {
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

func (s *CfgService) listenCfgPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenCfgPatches",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenCfgPatches",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	go conn.Subscribe(s.sub.Chan, s.sub.ChanName)

	var c DevCfg
	for {
		select {
		case msg := <-s.sub.Chan:
			if err := json.Unmarshal(msg, &c); err != nil {
				s.log.WithFields(logrus.Fields{
					"func": "listenCfgPatches",
				}).Errorf("%s", err)
			} else {
				go s.pubNewCfgPatchEvent(&c)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *CfgService) pubNewCfgPatchEvent(devCfg *DevCfg) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "pubNewCfgPatchEvent",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubNewCfgPatchEvent",
		}).Error("nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
	}
	defer conn.Close()

	e := proto.Event{
		AggregateId:   devCfg.MAC,
		AggregateType: aggregate,
		EventId:       uuid.NewV4().String(),
		EventType:     event,
		EventData:     string(devCfg.Data),
	}
	subj := "CfgService.Patch." + devCfg.MAC
	b, err := gproto.Marshal(&e)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubNewCfgPatchEvent",
		}).Errorf("marshal has failed: %s", err)
	}

	if err := conn.Publish(subj, b); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubNewCfgPatchEvent",
		}).Errorf("Publish has failed: %s", err)
	}

	s.log.WithFields(logrus.Fields{
		"func":  "pubNewCfgPatchEvent",
		"event": cfg.EventCfgPatchCreated,
	}).Infof("cfg patch [%s] for device with ID [%s]", devCfg.Data, devCfg.MAC)
}

func (s *CfgService) runConsulAgent() {
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

// todo: substitute bool with byte
func (s *CfgService) check() (bool, error) {
	// while the service is alive - everything is ok
	return true, nil
}

func (s *CfgService) updateTTL(check func() (bool, error)) {
	t := time.NewTicker(s.ttl / 2)
	for range t.C {
		s.update(check)
	}
}

func (s *CfgService) update(check func() (bool, error)) {
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

// SetDevInitCfg check's whether device is already registered in the system. If it's already registered,
// the func returns actual configuration. Otherwise it returns default config for that type of device.
func (s *CfgService) SetDevInitCfg(meta *DevMeta) (*DevCfg, error) {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	if err = conn.SetDevMeta(meta); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("%s", err)
		return nil, err
	}

	var devCfg *DevCfg
	id := DevID(meta.MAC)

	if ok, err := conn.DevIsRegistered(meta); ok {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}

		devCfg, err = conn.GetDevCfg(id)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}

		devCfg, err = conn.GetDevDefaultCfg(meta)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}

		if err = conn.SetDevCfg(id, devCfg); err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}
		s.log.WithFields(logrus.Fields{
			"func":  "SetDevInitCfg",
			"event": cfg.EventDevRegistered,
		}).Infof("devices' meta: %+v", meta)
	}
	return devCfg, err
}

// GetDevCfg returns configuration for the given device.
func (s *CfgService) GetDevCfg(id DevID) (*DevCfg, error) {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "GetDevCfg",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	c, err := s.store.GetDevCfg(id)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "GetDevCfg",
		}).Errorf("%s", err)
		return nil, err
	}
	return c, nil
}

// SetDevCfg sets configuration for the given device.
func (s *CfgService) SetDevCfg(id DevID, c *DevCfg) error {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevCfg",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	if err := conn.SetDevCfg(id, c); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevCfg",
		}).Errorf("%s", err)
		return err
	}
	return nil
}

// Publish posts a message on the given channel.
func (s *CfgService) PublishCfgPatch(c *DevCfg, channel string) (int64, error) {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "PublishCfgPatch",
		}).Errorf("%s", err)
		return 0, err
	}
	defer conn.CloseConn()

	numberOfClients, err := s.store.Publish(c, channel)
	if err != nil {
		return 0, err
	}
	return numberOfClients, nil
}
