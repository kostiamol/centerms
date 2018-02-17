package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/kostiamol/centerms/entities"
)

type DataService struct {
	Server  entities.Address
	Storage entities.Storage
	Ctrl    entities.ServiceController
	Log     *logrus.Entry
	PubSubj string
}

func NewDataService(srv entities.Address, storage entities.Storage, ctrl entities.ServiceController,
	log *logrus.Entry, subj string) *DataService {

	return &DataService{
		Server:  srv,
		Storage: storage,
		Ctrl:    ctrl,
		Log:     log.WithFields(logrus.Fields{"service": "data"}),
		PubSubj: subj,
	}
}

func (s *DataService) Run() {
	s.Log.WithFields(logrus.Fields{
		"func":  "Run",
		"event": "start",
	}).Infof("running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)

	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "Run",
				"event": "panic",
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
		case <-s.Ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *DataService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "terminate",
				"event": "panic",
			}).Errorf("%s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.WithFields(logrus.Fields{
		"func":  "terminate",
		"event": "service_terminated",
	}).Infoln("DataService is down")
	s.Ctrl.Terminate()
}

func (s *DataService) SaveDevData(req *entities.SaveDevDataRequest) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(req); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return
	}

	go s.publishDevData(req)
}

func (s *DataService) publishDevData(req *entities.SaveDevDataRequest) error {
	defer func() {
		if r := recover(); r != nil {
			s.Log.WithFields(logrus.Fields{
				"func":  "publishDevData",
				"event": "panic",
			}).Errorf("%s", r)
			s.terminate()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}
	defer conn.CloseConn()

	b, err := json.Marshal(req)
	if err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}

	if _, err = conn.Publish(s.PubSubj, b); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
