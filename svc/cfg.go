// Package svc provides definitions for services that run on the center.
package svc

import (
	"strconv"

	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/proto"
	"golang.org/x/net/context"

	"math/rand"
	"time"

	"encoding/json"

	gproto "github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	aggregate = "cfg_svc"
	event     = "cfg_patched"
)

type (
	// CfgService is used to deal with device configurations.
	CfgService struct {
		log      *logrus.Entry
		ctrl     Ctrl
		store    cfgStorer
		sub      subscription
		retry    time.Duration
		natsAddr cfg.Addr
	}

	// CfgServiceCfg is used to initialize an instance of CfgService.
	CfgServiceCfg struct {
		Log      *logrus.Entry
		Ctrl     Ctrl
		Store    cfgStorer
		SubChan  string
		Retry    time.Duration
		NATSAddr cfg.Addr
	}

	cfgStorer interface {
		SetDevCfg(id string, c *DevCfg) error
		GetDevCfg(id string) (*DevCfg, error)
		GetDevDefaultCfg(*DevMeta) (*DevCfg, error)
		SetDevMeta(*DevMeta) error
		DevIsRegistered(*DevMeta) (bool, error)
		Publish(msg interface{}, channel string) (int64, error)
		Subscribe(c chan []byte, channel ...string) error
	}

	// DevCfg holds device's MAC address and config.
	DevCfg struct {
		MAC  string          `json:"mac"`
		Data json.RawMessage `json:"data"`
	}

	subscription struct {
		ChanName string
		Chan     chan []byte
	}
)

// NewCfgService creates and initializes a new instance of CfgService.
func NewCfgService(c *CfgServiceCfg) *CfgService {
	return &CfgService{
		log:   c.Log.WithFields(logrus.Fields{"component": "cfg"}),
		ctrl:  c.Ctrl,
		store: c.Store,
		sub: subscription{
			ChanName: c.SubChan,
			Chan:     make(chan []byte),
		},
		retry:    c.Retry,
		natsAddr: c.NATSAddr,
	}
}

// Run launches the service by running goroutines for listening the service termination and config patches.
func (s *CfgService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Infof("is running")

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
}

func (s *CfgService) listenTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *CfgService) terminate() {
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

	go s.store.Subscribe(s.sub.Chan, s.sub.ChanName) // nolint

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

	conn, err := nats.Connect(s.natsAddr.Host + strconv.FormatUint(s.natsAddr.Port, 10))
	for err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubNewCfgPatchEvent",
		}).Error("nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "pubNewCfgPatchEvent",
			}).Errorf("Connect(): %s", err)
		}
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

// SetDevInitCfg check's whether device is already registered in the system. If it's already registered,
// the func returns actual configuration. Otherwise it returns default config for that type of device.
func (s *CfgService) SetDevInitCfg(meta *DevMeta) (*DevCfg, error) {
	if err := s.store.SetDevMeta(meta); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("%s", err)
		return nil, err
	}

	var devCfg *DevCfg
	id := meta.MAC

	if ok, err := s.store.DevIsRegistered(meta); ok {
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}

		devCfg, err = s.store.GetDevCfg(id)
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

		devCfg, err = s.store.GetDevDefaultCfg(meta)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"func": "SetDevInitCfg",
			}).Errorf("%s", err)
			return nil, err
		}

		if err = s.store.SetDevCfg(id, devCfg); err != nil {
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
	return devCfg, nil
}

// GetDevCfg returns configuration for the given device.
func (s *CfgService) GetDevCfg(id string) (*DevCfg, error) {
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
func (s *CfgService) SetDevCfg(id string, c *DevCfg) error {
	if err := s.store.SetDevCfg(id, c); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SetDevCfg",
		}).Errorf("%s", err)
		return err
	}
	return nil
}

// PublishCfgPatch posts a message on the given channel.
func (s *CfgService) PublishCfgPatch(c *DevCfg, channel string) (int64, error) {
	numberOfClients, err := s.store.Publish(c, channel)
	if err != nil {
		return 0, err
	}
	return numberOfClients, nil
}
