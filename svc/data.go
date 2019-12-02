package svc

import (
	"github.com/kostiamol/centerms/metric"
	"github.com/kostiamol/centerms/store/dev"

	"github.com/kostiamol/centerms/log"

	"golang.org/x/net/context"
)

type (
	// DataStorer is a contract for the data storer.
	DataStorer interface {
		GetDevsData() ([]dev.Data, error)
		GetDevData(id string) (*dev.Data, error)
		SaveData(*dev.Data) error
	}

	// Publisher .
	DataPublisher interface {
		Publish(msg interface{}, channel string) (int64, error)
	}

	// DataServiceCfg is used to initialize an instance of dataService.
	DataServiceCfg struct {
		Log     log.Logger
		Ctrl    Ctrl
		Metric  *metric.Metric
		Store   DataStorer
		PubChan chan<- *dev.Data
	}

	// dataService is used to deal with device data.
	dataService struct {
		log     log.Logger
		ctrl    Ctrl
		metric  *metric.Metric
		storer  DataStorer
		pubChan chan<- *dev.Data
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
	s.log.With("event", log.EventComponentStarted).Infof("")

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.With("event", log.EventPanic).Errorf("func Run: %s", r)
			s.metric.ErrorCounter(log.EventPanic)
			cancel()
			s.ctrl.Terminate()
		}
	}()

	go s.listenToTermination()
}

func (s *dataService) listenToTermination() {
	<-s.ctrl.StopChan
	s.log.With("event", log.EventComponentShutdown).Infof("")
	_ = s.log.Flush()
}

// SaveData is used to save device data to the store.
func (s *dataService) SaveData(d *dev.Data) error {
	if err := s.storer.SaveData(d); err != nil {
		s.log.Errorf("func SaveDevData: %s", err)
		return err
	}

	s.pubChan <- d

	return nil
}

// GetDevData is used to get device data from the store.
func (s *dataService) GetDevData(id string) (*dev.Data, error) {
	d, err := s.storer.GetDevData(id)
	if err != nil {
		s.log.Errorf("func GetDevData: %s", err)
		return nil, err
	}
	return d, nil
}

// GetDevsData is used to get all the devices data from the store.
func (s *dataService) GetDevsData() ([]dev.Data, error) {
	d, err := s.storer.GetDevsData()
	if err != nil {
		s.log.Errorf("func GetDevsData: %s", err)
		return nil, err
	}
	return d, nil
}
