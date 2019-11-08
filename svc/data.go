package svc

import (
	"encoding/json"

	"github.com/kostiamol/centerms/metric"

	"github.com/kostiamol/centerms/log"

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
		Log     log.Logger
		Ctrl    Ctrl
		Metric  *metric.Metric
		Store   DataStorer
		PubChan chan<- *DevData
	}

	// dataService is used to deal with device data.
	dataService struct {
		log     log.Logger
		ctrl    Ctrl
		metric  *metric.Metric
		storer  DataStorer
		pubChan chan<- *DevData
	}
)

// NewDataService creates and initializes a new instance of dataService.
func NewDataService(c *DataServiceCfg) *dataService { //nolint
	return &dataService{
		log:     c.Log.With("component", "data"),
		ctrl:    c.Ctrl,
		metric:  c.Metric,
		storer:  c.Store,
		pubChan: c.PubChan,
	}
}

// Run launches the service by running goroutine that listens to the service termination.
func (s *dataService) Run() {
	s.log.With("event", log.EventComponentStarted)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func Run: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
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
	s.log.With("event", log.EventComponentShutdown)
	_ = s.log.Flush()
	s.ctrl.Terminate()
}

// SaveDevData is used to save device data to the store.
func (s *dataService) SaveDevData(d *DevData) error {
	if err := s.storer.SaveDevData(d); err != nil {
		s.log.Errorf("func SaveDevData: %s", err)
		return err
	}

	s.pubChan <- d

	return nil
}

// GetDevData is used to get device data from the store.
func (s *dataService) GetDevData(id string) (*DevData, error) {
	d, err := s.storer.GetDevData(id)
	if err != nil {
		s.log.Errorf("func GetDevData: %s", err)
		return nil, err
	}
	return d, nil
}

// GetDevsData is used to get all the devices data from the store.
func (s *dataService) GetDevsData() ([]DevData, error) {
	d, err := s.storer.GetDevsData()
	if err != nil {
		s.log.Errorf("func GetDevsData: %s", err)
		return nil, err
	}
	return d, nil
}
