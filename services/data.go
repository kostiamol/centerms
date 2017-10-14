package services

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/pb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"os"
)

type DataService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
}

func NewDataService(s entities.Server, ds entities.DevStorage, c entities.ServicesController,
	l *logrus.Logger, r *time.Ticker) *DataService {
	l.Out = os.Stdout
	return &DataService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
	}
}

func (s *DataService) Run() {
	s.Log.Infof("DataService has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: Run(): panic(): %s", r)
			cancel()
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		s.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				s.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
			}
		}
		s.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, s)
	gs.Serve(ln)
}

func (s *DataService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *DataService) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DataService is down")
	s.Controller.Terminate()
}

func (s *DataService) SaveDevData(ctx context.Context, r *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	req := entities.Request{
		Time: r.Time,
		Meta: entities.DevMeta{
			Type: r.Meta.Type,
			Name: r.Meta.Name,
			MAC:  r.Meta.Mac,
		},
		Data: r.Data,
	}
	s.saveDevData(&req)
	return &pb.SaveDevDataResponse{Status: "OK"}, nil
}

func (s *DataService) SetDevInitConfig(ctx context.Context, r *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	return nil, nil
}

func (s *DataService) saveDevData(r *entities.Request) {
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

func (s *DataService) publishDevData(r *entities.Request, channel string) error {
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
