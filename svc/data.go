package svc

import (
	"encoding/json"

	"go.uber.org/zap"

	"github.com/kostiamol/centerms/cfg"

	"golang.org/x/net/context"
)

type (
	// DataService is used to deal with device data.
	DataService struct {
		log     *zap.SugaredLogger
		ctrl    Ctrl
		store   dataStorer
		pubChan string
	}

	// DataServiceCfg is used to initialize an instance of DataService.
	DataServiceCfg struct {
		Log     *zap.SugaredLogger
		Ctrl    Ctrl
		Store   dataStorer
		PubChan string
	}

	dataStorer interface {
		GetDevsData() ([]DevData, error)
		GetDevData(id string) (*DevData, error)
		SaveDevData(*DevData) error
		Publish(msg interface{}, channel string) (int64, error)
	}

	// DevMeta is used to subscriber device metadata: it's type, name (model) and MAC address.
	DevMeta struct {
		Type string `json:"type"`
		Name string `json:"name"`
		MAC  string `json:"mac"`
	}

	// DevData is used to subscriber time of the request, device's metadata and the data it transfers.
	DevData struct {
		Time int64           `json:"time"`
		Meta DevMeta         `json:"meta"`
		Data json.RawMessage `json:"data"`
	}
)

// NewDataService creates and initializes a new instance of DataService.
func NewDataService(c *DataServiceCfg) *DataService {
	return &DataService{
		log:     c.Log.With("component", "data"),
		ctrl:    c.Ctrl,
		store:   c.Store,
		pubChan: c.PubChan,
	}
}

// Run launches the service by running goroutine that listens for the service termination.
func (s *DataService) Run() {
	s.log.With("event", cfg.EventComponentStarted).Info("is running")

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("func", "Run", "event", cfg.EventPanic).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()
	go s.listenTermination()
}

func (s *DataService) listenTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *DataService) terminate() {
	s.log.With("event", cfg.EventComponentShutdown).Info("svc is down")
	s.log.Sync() // nolint
	s.ctrl.Terminate()
}

// SaveDevData is used to save device data to the store.
func (s *DataService) SaveDevData(data *DevData) error {
	if err := s.store.SaveDevData(data); err != nil {
		s.log.Errorf("SaveDevData(): %s", err)
		return err
	}
	go s.pubDevData(data) // nolint
	return nil
}

func (s *DataService) pubDevData(data *DevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", cfg.EventPanic).Errorf("pubDevData(): %s", r)
			s.terminate()
		}
	}()
	b, err := json.Marshal(data)
	if err != nil {
		s.log.Errorf("pubDevData(): %s", err)
		return err
	}
	if _, err = s.store.Publish(b, s.pubChan); err != nil {
		s.log.Errorf("pubDevData(): %s", err)
		return err
	}
	return nil
}

// GetDevData is used to get device data from the store.
func (s *DataService) GetDevData(id string) (*DevData, error) {
	d, err := s.store.GetDevData(id)
	if err != nil {
		s.log.Errorf("GetDevData(): %s", err)
		return nil, err
	}
	return d, nil
}

// GetDevsData is used to get all the devices data from the store.
func (s *DataService) GetDevsData() ([]DevData, error) {
	ds, err := s.store.GetDevsData()
	if err != nil {
		s.log.Errorf("GetDevsData(): %s", err)
		return nil, err
	}
	return ds, nil
}
