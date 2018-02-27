package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/kostiamol/centerms/entities"
)

type DataService struct {
	Server  entities.Address
	Storage entities.Storager
	Ctrl    entities.ServiceController
	Log     *logrus.Entry
	PubChan string
}

func NewDataService(srv entities.Address, storage entities.Storager, ctrl entities.ServiceController,
	log *logrus.Entry, pubChan string) *DataService {

	return &DataService{
		Server:  srv,
		Storage: storage,
		Ctrl:    ctrl,
		Log:     log.WithFields(logrus.Fields{"service": "data"}),
		PubChan: pubChan,
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

func (s *DataService) SaveDevData(req *entities.SaveDevDataReq) {
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

func (s *DataService) publishDevData(req *entities.SaveDevDataReq) error {
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
	if _, err = conn.Publish(b, s.PubChan); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "publishDevData",
		}).Errorf("%s", err)
		return err
	}

	return nil
}
