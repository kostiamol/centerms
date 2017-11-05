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
	Storage    entities.Storage
	Ctrl       entities.ServiceController
	Log        *logrus.Logger
	PubSubject string
}

func NewDataService(srv entities.Server, st entities.Storage, c entities.ServiceController,
	l *logrus.Logger, subj string) *DataService {

	l.Out = os.Stdout
	return &DataService{
		Server:     srv,
		Storage:    st,
		Ctrl:       c,
		Log:        l,
		PubSubject: subj,
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
		case <-s.Ctrl.StopChan:
			s.terminate()
			return
		}
	}
}

func (s *DataService) terminate() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: terminate(): panic(): %s", r)
			s.Ctrl.Terminate()
		}
	}()

	s.Storage.CloseConn()
	s.Log.Infoln("DataService is down")
	s.Ctrl.Terminate()
}

func (s *DataService) SaveDevData(req *entities.SaveDevDataRequest) {
	conn, err := s.Storage.CreateConn()
	if err != nil {
		s.Log.Errorf("DataService: saveDevData(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(req); err != nil {
		s.Log.Errorf("DataService: SaveDevData() has failed: %s", err)
		return
	}

	go s.publishDevData(req)
}

func (s *DataService) publishDevData(req *entities.SaveDevDataRequest) error {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: publishDevData(): panic(): %s", r)
			s.terminate()
		}
	}()

	conn, err := s.Storage.CreateConn()
	if err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): Request marshalling has failed")
	}

	if _, err = conn.Publish(s.PubSubject, b); err != nil {
		return errors.Wrap(err, "DataService: publishDevData(): publishing has failed")
	}

	return nil
}
