// Package svc provides definitions for services that run on the center.
package svc

import (
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/log"
	"golang.org/x/net/context"

	"encoding/json"
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

	// Subscriber .
	Subscriber interface {
		Subscribe(c chan []byte, channel ...string) error
	}

	// Publisher .
	Publisher interface {
		Publish(mac, data string) error
	}

	// DevCfg holds device's MAC address and config.
	DevCfg struct {
		MAC  string          `json:"mac"`
		Data json.RawMessage `json:"data"`
	}

	// CfgServiceCfg is used to initialize an instance of cfgService.
	CfgServiceCfg struct {
		Log        log.Logger
		Ctrl       Ctrl
		Store      CfgStorer
		Subscriber Subscriber
		Publisher  Publisher
		SubChan    string
	}

	// cfgService is used to deal with device configurations.
	cfgService struct {
		log          log.Logger
		ctrl         Ctrl
		storer       CfgStorer
		subscriber   Subscriber
		subscription subscription
		publisher    Publisher
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
		publisher: c.Publisher,
	}
}

// Run launches the service by running goroutines for listening to the service termination and config patches.
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

	go s.listenToTermination()
	go s.subscribeToCfgPatches()
	go s.listenToCfgPatches(ctx)
}

func (s *cfgService) listenToTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *cfgService) terminate() {
	s.log.With("event", cfg.EventComponentShutdown).Info("is down")
	_ = s.log.Flush()
	s.ctrl.Terminate()
}

func (s *cfgService) subscribeToCfgPatches() {
	if err := s.subscriber.Subscribe(s.subscription.Chan, s.subscription.ChanName); err != nil {
		s.log.Errorf("Subscribe(): %s", err)
	}
}

func (s *cfgService) listenToCfgPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("listenToCfgPatches(): %s", r)
			s.terminate()
		}
	}()

	var c DevCfg
	for {
		select {
		case msg := <-s.subscription.Chan:
			if err := json.Unmarshal(msg, &c); err != nil {
				s.log.Errorf("listenToCfgPatches(): %s", err)
			} else {
				if err := s.publisher.Publish(c.MAC, string(c.Data)); err != nil {
					s.log.Errorf("Publish: %s", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
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
