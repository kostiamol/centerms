package svc

import (
	"encoding/json"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/cfg"

	"golang.org/x/net/context"
)

type (
	// DataStorer is a contract for the data storer.
	DataStorer interface {
		GetDevsData() ([]DevData, error)
		GetDevData(id string) (*DevData, error)
		SaveDevData(*DevData) error
	}

	// Publisher .
	DataPublisher interface {
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

	// DataServiceCfg is used to initialize an instance of dataService.
	DataServiceCfg struct {
		Log       log.Logger
		Ctrl      Ctrl
		Store     DataStorer
		Publisher DataPublisher
		PubChan   string
	}

	// dataService is used to deal with device data.
	dataService struct {
		log       log.Logger
		ctrl      Ctrl
		storer    DataStorer
		publisher DataPublisher
		pubChan   string
	}
)

// NewDataService creates and initializes a new instance of dataService.
func NewDataService(c *DataServiceCfg) *dataService { //nolint
	return &dataService{
		log:       c.Log.With("component", "data"),
		ctrl:      c.Ctrl,
		storer:    c.Store,
		publisher: c.Publisher,
		pubChan:   c.PubChan,
	}
}

// Run launches the service by running goroutine that listens to the service termination.
func (s *dataService) Run() {
	s.log.With("event", cfg.EventComponentStarted).Info("is running")

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("func", "Run", "event", cfg.EventPanic).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenToTermination()
}

func (s *dataService) listenToTermination() {
	<-s.ctrl.StopChan
	s.terminate()
}

func (s *dataService) terminate() {
	s.log.With("event", cfg.EventComponentShutdown).Info("svc is down")
	_ = s.log.Flush()
	s.ctrl.Terminate()
}

// SaveDevData is used to save device data to the store.
func (s *dataService) SaveDevData(d *DevData) error {
	if err := s.storer.SaveDevData(d); err != nil {
		s.log.Errorf("SaveDevData(): %s", err)
		return err
	}

	b, err := json.Marshal(d)
	if err != nil {
		s.log.Errorf("Marshal(): %s", err)
		return err
	}

	if _, err = s.publisher.Publish(b, s.pubChan); err != nil {
		s.log.Errorf("Publish(): %s", err)
		return err
	}

	return nil
}

// GetDevData is used to get device data from the store.
func (s *dataService) GetDevData(id string) (*DevData, error) {
	d, err := s.storer.GetDevData(id)
	if err != nil {
		s.log.Errorf("GetDevData(): %s", err)
		return nil, err
	}
	return d, nil
}

// GetDevsData is used to get all the devices data from the store.
func (s *dataService) GetDevsData() ([]DevData, error) {
	d, err := s.storer.GetDevsData()
	if err != nil {
		s.log.Errorf("GetDevsData(): %s", err)
		return nil, err
	}
	return d, nil
}
