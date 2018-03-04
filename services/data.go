package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/kostiamol/centerms/entities"
)

type DataService struct {
	addr    entities.Address
	storage entities.Storager
	ctrl    entities.ServiceController
	log     *logrus.Entry
	pubChan string
}

func NewDataService(srv entities.Address, storage entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, pubChan string) *DataService {

	return &DataService{
		addr:    srv,
		storage: storage,
		ctrl:    ctrl,
		log:     log.WithFields(logrus.Fields{"service": "data"}),
		pubChan: pubChan,
	}
}

func (s *DataService) Run() {
	s.log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.addr.Host, s.addr.Port)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": "panic",
			}).Errorf("%s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
}

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
				"event": "panic",
			}).Errorf("%s", r)
			s.ctrl.Terminate()
		}
	}()

	s.storage.CloseConn()
	s.log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Infoln("DataService is down")
	s.ctrl.Terminate()
}

func (s *DataService) SaveDevData(data *entities.RawDevData) {
	conn, err := s.storage.CreateConn()
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

	go s.publishDevData(data)
}

func (s *DataService) publishDevData(data *entities.RawDevData) error {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "publishDevData",
				"event": "panic",
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

	b, err := json.Marshal(data)
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
