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
)

type DevDataService struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServicesController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
}

func NewDevDataService(s entities.Server, ds entities.DevStorage, c entities.ServicesController,
	l *logrus.Logger, r *time.Ticker) *DevDataService {
	return &DevDataService{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
	}
}

func (s *DevDataService) Run() {
	s.Log.Infof("DevDataService has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DevDataService: Run(): panic(): %s", r)
			cancel()
			s.handleTermination()
		}
	}()

	go s.listenTermination()

	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		s.Log.Errorf("DevDataService: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				s.Log.Errorf("DevDataService: Run(): Listen() has failed: %s", err)
			}
		}
		s.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, s)
	gs.Serve(ln)
}

func (s *DevDataService) listenTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.handleTermination()
			return
		}
	}
}

func (s *DevDataService) handleTermination() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DevDataService is down")
	s.Controller.Terminate()
}

func (s *DevDataService) SaveDevData(ctx context.Context, r *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	req := entities.Request{
		Time:   r.Time,
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

func (s *DevDataService) SetDevInitConfig(ctx context.Context, r *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	return nil, nil
}

func (s *DevDataService) saveDevData(r *entities.Request) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevDataService: saveDevData(): storage connection hasn't been established: %s", err)
		return
	}
	defer conn.CloseConn()

	if err = conn.SaveDevData(r); err != nil {
		s.Log.Errorf("DevDataService: SaveDevData() has failed: %s", err)
		return
	}

	//s.Log.Infof("save data for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	go s.publishDevData(r, entities.DevDataChan)
}

func (s *DevDataService) publishDevData(r *entities.Request, channel string) error {
	b, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "DevDataService: publishDevData(): Request marshalling has failed")
	}

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		return errors.Wrap(err, "DevDataService: publishDevData(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	if _, err = conn.Publish(channel, b); err != nil {
		return errors.Wrap(err, "DevDataService: publishDevData(): publishing has failed")
	}

	//s.Log.Infof("publish DevData for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	return nil
}

