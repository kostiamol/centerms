package grpcsvc

import (
	"math/rand"
	"time"

	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/pb"
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

func newCenterServiceGRPC(c GRPCConfig) *api {
	return &api{
		configService: *c.ConfigService,
		dataService:   *c.DataService,
		retry:         c.Retry,
		log:           c.Log.WithFields(logrus.Fields{"service": "grpc"}),
	}
}

type api struct {
	configService services.ConfigService
	dataService   services.DataService
	ctrl          entities.ServiceController
	retry         time.Duration
	log           *logrus.Entry
}

// SetDevInitConfig sets device's initial config when it connects to the center for the first time using ConfigService
// and returns that config to the device.
func (a *api) SetDevInitConfig(ctx context.Context, req *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse,
	error) {
	meta := entities.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	config, _ := a.configService.SetDevInitConfig(&meta)

	return &pb.SetDevInitConfigResponse{
		Config: config.Data,
	}, nil
}

// SaveDevData saves data from device using DataService.
func (a *api) SaveDevData(ctx context.Context, req *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	data := entities.RawDevData{
		Time: req.Time,
		Meta: entities.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	a.dataService.SaveDevData(&data)

	return &pb.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (a *api) listenConfig() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenConfig",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			a.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.configService.GetAddr().Host+":"+fmt.Sprint(a.configService.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.configService.GetAddr().Host+":"+fmt.Sprint(a.configService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	if gs.Serve(ln); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}

func (a *api) listenData() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": entities.EventPanic,
			}).Errorf("%s", r)
			a.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.dataService.GetAddr().Host+":"+fmt.Sprint(a.dataService.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.dataService.GetAddr().Host+":"+fmt.Sprint(a.dataService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	if gs.Serve(ln); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}
