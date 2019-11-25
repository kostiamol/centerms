// Package svc provides definitions for services that run on the center.
package svc

import (
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/metric"
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
		Log       log.Logger
		Ctrl      Ctrl
		Metric    *metric.Metric
		Store     CfgStorer
		Publisher Publisher
		SubChan   <-chan *DevCfg
	}

	// cfgService is used to deal with device configurations.
	cfgService struct {
		log       log.Logger
		ctrl      Ctrl
		metric    *metric.Metric
		storer    CfgStorer
		publisher Publisher
		subChan   <-chan *DevCfg
	}
)

// NewCfgService creates and initializes a new instance of cfgService.
func NewCfgService(c *CfgServiceCfg) *cfgService { // nolint
	return &cfgService{
		log:       c.Log.With("component", "cfg"),
		ctrl:      c.Ctrl,
		metric:    c.Metric,
		storer:    c.Store,
		publisher: c.Publisher,
		subChan:   c.SubChan,
	}
}

// Run launches the service by running goroutines for listening to the service termination and config patches.
func (s *cfgService) Run() {
	s.log.With("event", log.EventComponentStarted).Infof("")

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func Run: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
			cancel()
			s.ctrl.Terminate()
		}
	}()

	go s.listenToTermination()
	go s.listenToCfgPatches(ctx)
}

func (s *cfgService) listenToTermination() {
	<-s.ctrl.StopChan
	s.log.With("event", log.EventComponentShutdown).Infof("")
	_ = s.log.Flush()
}

func (s *cfgService) listenToCfgPatches(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func listenToCfgPatches: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
			s.ctrl.Terminate()
		}
	}()

	for {
		select {
		case msg := <-s.subChan:
			if err := s.publisher.Publish(msg.MAC, string(msg.Data)); err != nil {
				s.log.Errorf("func Publish: %s", err)
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
		s.log.Errorf("func SetDevInitCfg: %s", err)
		return nil, err
	}

	var devCfg *DevCfg
	id := meta.MAC

	if ok, err := s.storer.DevIsRegistered(meta); ok {
		if err != nil {
			s.log.Errorf("func SetDevInitCfg: %s", err)
			return nil, err
		}

		devCfg, err = s.storer.GetDevCfg(id)
		if err != nil {
			s.log.Errorf("func SetDevInitCfg: %s", err)
			return nil, err
		}
	} else {
		if err != nil {
			s.log.Errorf("func SetDevInitCfg: %s", err)
			return nil, err
		}

		devCfg, err = s.storer.GetDevDefaultCfg(meta)
		if err != nil {
			s.log.Errorf("func SetDevInitCfg: %s", err)
			return nil, err
		}

		if err = s.storer.SetDevCfg(id, devCfg); err != nil {
			s.log.Errorf("func SetDevInitCfg: %s", err)
			return nil, err
		}
		s.log.With("event", log.EventDevRegistered).Infof("meta: %+v", meta)
	}
	return devCfg, nil
}

// GetDevCfg returns configuration for the given device.
func (s *cfgService) GetDevCfg(id string) (*DevCfg, error) {
	c, err := s.storer.GetDevCfg(id)
	if err != nil {
		s.log.Errorf("func GetDevCfg: %s", err)
		return nil, err
	}
	return c, nil
}

// SetDevCfg sets configuration for the given device.
func (s *cfgService) SetDevCfg(id string, c *DevCfg) error {
	if err := s.storer.SetDevCfg(id, c); err != nil {
		s.log.Errorf("func SetDevCfg: %s", err)
		return err
	}
	return nil
}
