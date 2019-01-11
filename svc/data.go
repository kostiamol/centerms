package svc

import (
	"encoding/json"

	"github.com/kostiamol/centerms/cfg"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
)

// DevMeta is used to store device metadata: it's type, name (model) and MAC address.
type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

// DevData is used to store time of the request, device's metadata and the data it transfers.
type DevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// DataService is used to deal with device data.
type DataService struct {
	store   Storer
	ctrl    Ctrl
	log     *logrus.Entry
	pubChan string
}

// DataServiceCfg is used to initialize an instance of DataService.
type DataServiceCfg struct {
	Store   Storer
	Ctrl    Ctrl
	Log     *logrus.Entry
	PubChan string
}

// NewDataService creates and initializes a new instance of DataService.
func NewDataService(c *DataServiceCfg) *DataService {
	return &DataService{
		store:   c.Store,
		ctrl:    c.Ctrl,
		log:     c.Log.WithFields(logrus.Fields{"component": "data"}),
		pubChan: c.PubChan,
	}
}

// Run launches the service by running goroutine that listens for the service termination.
func (s *DataService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": cfg.EventSVCStarted,
	}).Info("is running")

	_, cancel := context.WithCancel(context.Background())
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
}

func (s *DataService) listenTermination() {
	for {
		select {
		case <-s.ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *DataService) terminate() {
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

// SaveDevData is used to save device data in the store.
func (s *DataService) SaveDevData(data *DevData) {
	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	go s.pubDevData(data)
}

func (s *DataService) pubDevData(data *DevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "pubDevData",
				"event": cfg.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.store.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(data)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "pubDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
