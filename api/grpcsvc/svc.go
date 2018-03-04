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

type GRPCConfig struct {
	ConfigService *services.ConfigService
	DataService   *services.DataService
	Retry         time.Duration
	Log           *logrus.Entry
}

func Init(c GRPCConfig) {
	s := newCenterServiceGRPC(c)
	go s.listenConfig()
	go s.listenData()
}

func newCenterServiceGRPC(c GRPCConfig) *API {
	return &API{
		configService: *c.ConfigService,
		dataService:   *c.DataService,
		retry:         c.Retry,
		log:           c.Log.WithFields(logrus.Fields{"service": "grpc"}),
	}
}

type API struct {
	configService services.ConfigService
	dataService   services.DataService
	ctrl          entities.ServiceController
	retry         time.Duration
	log           *logrus.Entry
}

func (api *API) SetDevInitConfig(ctx context.Context, req *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	meta := entities.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	config, _ := api.configService.SetDevInitConfig(&meta)

	return &pb.SetDevInitConfigResponse{
		Config: config.Data,
	}, nil
}

func (api *API) SaveDevData(ctx context.Context, req *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	data := entities.RawDevData{
		Time: req.Time,
		Meta: entities.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	api.dataService.SaveDevData(&data)

	return &pb.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (api *API) listenConfig() {
	defer func() {
		if r := recover(); r != nil {
			api.log.WithFields(logrus.Fields{
				"func":  "listenConfig",
				"event": "panic",
			}).Errorf("%s", r)
			api.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", api.configService.GetAddr().Host+":"+fmt.Sprint(api.configService.GetAddr().Port))
	for err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(api.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", api.configService.GetAddr().Host+":"+fmt.Sprint(api.configService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, api)
	if gs.Serve(ln); err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}

func (api *API) listenData() {
	defer func() {
		if r := recover(); r != nil {
			api.log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": "panic",
			}).Errorf("%s", r)
			api.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", api.dataService.GetAddr().Host+":"+fmt.Sprint(api.dataService.GetAddr().Port))
	for err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(api.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", api.dataService.GetAddr().Host+":"+fmt.Sprint(api.dataService.GetAddr().Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, api)
	if gs.Serve(ln); err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}
