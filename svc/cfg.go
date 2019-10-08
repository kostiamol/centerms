// Package svc provides definitions for services that run on the center.
package svc

import (
	"fmt"
	"strconv"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/proto"
	"golang.org/x/net/context"

	"math/rand"
	"time"

	"encoding/json"

	gproto "github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/satori/go.uuid"
)

const (
	cfgPatchSubject = "cfg_patch"
	aggregate       = "cfg_svc"
	event           = "cfg_patched"
)

type (
	// CfgStorer is a contract for the configuration storer.
	CfgStorer interface {
		SetDevCfg(id string, c *DevCfg) error
		GetDevCfg(id string) (*DevCfg, error)
		GetDevDefaultCfg(*DevMeta) (*DevCfg, error)
		SetDevMeta(*DevMeta) error
		DevIsRegistered(*DevMeta) (bool, error)
	}

	// CfgSubscriber is a contract for the configuration subscriber.
	CfgSubscriber interface {
		Subscribe(c chan []byte, channel ...string) error
	}

	// CfgSubscriber is a contract for the configuration publisher.
	CfgPublisher interface {
		Publish(msg interface{}, channel string) (int64, error)
	}

	// DevCfg holds device's MAC address and config.
	DevCfg struct {
		MAC  string          `json:"mac"`
		Data json.RawMessage `json:"data"`
	}

	// CfgServiceCfg is used to initialize an instance of cfgService.
	CfgServiceCfg struct {
		Log           log.Logger
		Ctrl          Ctrl
		Store         CfgStorer
		Subscriber    CfgSubscriber
		Publisher     CfgPublisher
		SubChan       string
		NATSAddr      cfg.Addr
		RetryTimeout  time.Duration
		RetryAttempts uint64
	}

	// cfgService is used to deal with device configurations.
	cfgService struct {
		log           log.Logger
		ctrl          Ctrl
		storer        CfgStorer
		subscriber    CfgSubscriber
		subscription  subscription
		publisher     CfgPublisher
		natsAddr      cfg.Addr
		retryTimeout  time.Duration
		retryAttempts uint64
	}

	subscription struct {
		ChanName string
		Chan     chan []byte
	}
)

// NewCfgService creates and initializes a new instance of cfgService.
func NewCfgService(c *CfgServiceCfg) *cfgService { // nolint
	return &cfgService{
		log:        c.Log.With("component", "cfg"),
		ctrl:       c.Ctrl,
		storer:     c.Store,
		subscriber: c.Subscriber,
		subscription: subscription{
			ChanName: c.SubChan,
			Chan:     make(chan []byte),
		},
		publisher:     c.Publisher,
		natsAddr:      c.NATSAddr,
		retryTimeout:  c.RetryTimeout,
		retryAttempts: c.RetryAttempts,
	}
}

// Run launches the service by running goroutines for listening the service termination and config patches.
func (s *cfgService) Run() {
	s.log.With("event", cfg.EventComponentStarted).Infof("is running")

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("Run() %s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
	go s.listenCfgPatches(ctx)
}

func (s *cfgService) listenTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *cfgService) terminate() {
	s.log.With("event", cfg.EventComponentShutdown).Info("is down")
	_ = s.log.Flush()
	s.ctrl.Terminate()
}

func (s *cfgService) listenCfgPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("listenCfgPatches(): %s", r)
			s.terminate()
		}
	}()

	go s.subscriber.Subscribe(s.subscription.Chan, s.subscription.ChanName) // nolint

	var c DevCfg
	for {
		select {
		case msg := <-s.subscription.Chan:
			if err := json.Unmarshal(msg, &c); err != nil {
				s.log.Errorf("listenCfgPatches(): %s", err)
			} else {
				go s.pubNewCfgPatchEvent(&c)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *cfgService) pubNewCfgPatchEvent(devCfg *DevCfg) {
	var (
		err          error
		conn         *nats.Conn
		retryAttempt uint64
	)

	for {
		conn, err = nats.Connect(s.natsAddr.Host + strconv.FormatUint(s.natsAddr.Port, 10))
		if err != nil && retryAttempt < s.retryAttempts {
			s.log.Error("pubNewCfgPatchEvent(): nats connectivity status is DISCONNECTED")
			retryAttempt++
			duration := time.Duration(rand.Intn(int(s.retryTimeout.Seconds())))
			time.Sleep(time.Second*duration + 1)
			continue
		}
		break
	}
	defer conn.Close()

	e := proto.Event{
		AggregateId:   devCfg.MAC,
		AggregateType: aggregate,
		EventId:       uuid.NewV4().String(),
		EventType:     event,
		EventData:     string(devCfg.Data),
	}

	subj := fmt.Sprintf("%s.%s", cfgPatchSubject, devCfg.MAC)

	b, err := gproto.Marshal(&e)
	if err != nil {
		s.log.Errorf("pubNewCfgPatchEvent(): marshal failed: %s", err)
	}

	if err := conn.Publish(subj, b); err != nil {
		s.log.Errorf("pubNewCfgPatchEvent(): publish failed: %s", err)
	}

	s.log.With("func", "pubNewCfgPatchEvent", "event", cfg.EventCfgPatchCreated).
		Infof("cfg patch [%s] for device with ID [%s]", devCfg.Data, devCfg.MAC)
}

// SetDevInitCfg check's whether device is already registered in the system. If it's already registered,
// the func returns actual configuration. Otherwise it returns default config for that type of device.
func (s *cfgService) SetDevInitCfg(meta *DevMeta) (*DevCfg, error) {
	if err := s.storer.SetDevMeta(meta); err != nil {
		s.log.Errorf("SetDevInitCfg(): %s", err)
		return nil, err
	}

	var devCfg *DevCfg
	id := meta.MAC

	if ok, err := s.storer.DevIsRegistered(meta); ok {
		if err != nil {
			s.log.Errorf("SetDevInitCfg(): %s", err)
			return nil, err
		}

		devCfg, err = s.storer.GetDevCfg(id)
		if err != nil {
			s.log.Errorf("SetDevInitCfg(): %s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.log.Errorf("SetDevInitCfg(): %s", err)
			return nil, err
		}

		devCfg, err = s.storer.GetDevDefaultCfg(meta)
		if err != nil {
			s.log.Errorf("SetDevInitCfg(): %s", err)
			return nil, err
		}

		if err = s.storer.SetDevCfg(id, devCfg); err != nil {
			s.log.Errorf("SetDevInitCfg(): %s", err)
			return nil, err
		}
		s.log.With("func", "SetDevInitCfg", "event", cfg.EventDevRegistered).
			Infof("devices' meta: %+v", meta)
	}
	return devCfg, nil
}

// GetDevCfg returns configuration for the given device.
func (s *cfgService) GetDevCfg(id string) (*DevCfg, error) {
	c, err := s.storer.GetDevCfg(id)
	if err != nil {
		s.log.Errorf("GetDevCfg(): %s", err)
		return nil, err
	}
	return c, nil
}

// SetDevCfg sets configuration for the given device.
func (s *cfgService) SetDevCfg(id string, c *DevCfg) error {
	if err := s.storer.SetDevCfg(id, c); err != nil {
		s.log.Errorf("SetDevCfg(): %s", err)
		return err
	}
	return nil
}

// PublishCfgPatch posts a message on the given channel.
func (s *cfgService) PublishCfgPatch(c *DevCfg, channel string) (int64, error) {
	numberOfClients, err := s.publisher.Publish(c, channel)
	if err != nil {
		return 0, err
	}
	return numberOfClients, nil
}
