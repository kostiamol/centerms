package services

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"os"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type DataService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
}

func NewDataService(s entities.Server, ds entities.DevStorage, c entities.ServicesController,
	l *logrus.Logger) *DataService {

	l.Out = os.Stdout
	return &DataService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
	}
}

func (s *DataService) Run() {
	s.Log.Infof("DataService   is running on host: [%s], port: [%s]", s.Server.Host, s.Server.Port)
	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: Run(): panic(): %s", r)
			cancel()
			s.terminate()
		}
	}()

	go s.listenTermination()
}

func (s *DataService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *DataService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: terminate(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	s.DevStorage.CloseConn()
	s.Log.Infoln("DataService is down")
	s.Controller.Terminate()
}

func (s *DataService) SaveDevData(r *entities.SaveDevDataRequest) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DataService: saveDevData(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(r); err != nil {
		s.Log.Errorf("DataService: SaveDevData() has failed: %s", err)
		return
	}

	//s.Log.Infof("save data for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	go s.publishDevData(r, entities.DevDataChan)
}

func (s *DataService) publishDevData(r *entities.SaveDevDataRequest, channel string) error {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: publishDevData(): panic(): %s", r)
			s.terminate()
		}
	}()

	b, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): Request marshalling has failed")
	}

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	if _, err = conn.Publish(channel, b); err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): publishing has failed")
	}

	//s.Log.Infof("publish DevData for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	return nil
}
