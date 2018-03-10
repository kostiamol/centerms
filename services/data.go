package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/kostiamol/centerms/entities"
)

// DataService is used to deal with device data.
type DataService struct {
	addr    entities.Address
	storage entities.Storager
	ctrl    entities.ServiceController
	log     *logrus.Entry
	pubChan string
}

// NewDataService creates and initializes a new instance of DataService.
func NewDataService(srv entities.Address, st entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, pubChan string) *DataService {

	return &DataService{
		addr:    srv,
		storage: st,
		ctrl:    ctrl,
		log:     log.WithFields(logrus.Fields{"service": "data"}),
		pubChan: pubChan,
	}
}

// Run launches the service by running goroutine that listens for the service termination.
func (s *DataService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": entities.EventSVCStarted,
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
}

// GetAddr returns address of the service.
func (s *DataService) GetAddr() entities.Address {
	return s.addr
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
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": entities.EventSVCShutdown,
	}).Infoln("service is down")
	s.ctrl.Terminate()
}

// SaveDevData is used to save device data in the storage.
func (s *DataService) SaveDevData(d *entities.DevData) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(d); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}

	go s.publishDevData(d)
}

func (s *DataService) publishDevData(d *entities.DevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "publishDevData",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(d)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
