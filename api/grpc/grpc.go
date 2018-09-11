package grpc

import (
	"math/rand"
	"time"

	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GRPCConfig holds services for handling device data and configs, retry interval and log.
type GRPCConfig struct {
	ConfigService *services.ConfigService
	DataService   *services.DataService
	Retry         time.Duration
	Log           *logrus.Entry
}

// Init initializes gRPC and starts goroutines for listening device data and configs.
func Init(c GRPCConfig) {
	s := newCenterServiceGRPC(c)
	go s.listenConfig()
	go s.listenData()
}

func newCenterServiceGRPC(c GRPCConfig) *svc {
	return &svc{
		configService: *c.ConfigService,
		dataService:   *c.DataService,
		retry:         c.Retry,
		log:           c.Log.WithFields(logrus.Fields{"service": "api"}),
	}
}

type svc struct {
	configService services.ConfigService
	dataService   services.DataService
	ctrl          entities.ServiceController
	retry         time.Duration
	log           *logrus.Entry
}

// SetDevInitConfig sets device's initial config when it connects to the center for the first time using ConfigService
// and returns that config to the device.
func (s *svc) SetDevInitConfig(ctx context.Context, req *api.SetDevInitConfigRequest) (*api.SetDevInitConfigResponse,
	error) {
	meta := entities.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	config, _ := s.configService.SetDevInitConfig(&meta)

	return &api.SetDevInitConfigResponse{
		Config: config.Data,
	}, nil
}

// SaveDevData saves data from device using DataService.
func (s *svc) SaveDevData(ctx context.Context, req *api.SaveDevDataRequest) (*api.SaveDevDataResponse, error) {
	data := entities.DevData{
		Time: req.Time,
		Meta: entities.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	s.dataService.SaveDevData(&data)

	return &api.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (s *svc) listenConfig() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenConfig",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", s.configService.GetAddr().Host+":"+fmt.Sprint(s.configService.GetAddr().Port))
	for err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", s.configService.GetAddr().Host+":"+fmt.Sprint(s.configService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	api.RegisterCenterServiceServer(gs, s)
	if gs.Serve(ln); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}

func (s *svc) listenData() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			s.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", s.dataService.GetAddr().Host+":"+fmt.Sprint(s.dataService.GetAddr().Port))
	for err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", s.dataService.GetAddr().Host+":"+fmt.Sprint(s.dataService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	api.RegisterCenterServiceServer(gs, s)
	if gs.Serve(ln); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}
