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
	RetryInterval time.Duration
	Log           *logrus.Entry
}

func Init(c GRPCConfig) {
	s := newCenterServiceGRPC(c)
	go s.listenConfig()
	go s.listenData()
}

func newCenterServiceGRPC(c GRPCConfig) *API {
	return &API{
		Config:        *c.ConfigService,
		Data:          *c.DataService,
		RetryInterval: c.RetryInterval,
		Log:           c.Log.WithFields(logrus.Fields{"service": "grpc"}),
	}
}

type API struct {
	Config        services.ConfigService
	Data          services.DataService
	RetryInterval time.Duration
	Log           *logrus.Entry
}

func (api *API) SetDevInitConfig(ctx context.Context, req *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	meta := entities.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	config, _ := api.Config.SetDevInitConfig(&meta)

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

	api.Data.SaveDevData(&data)

	return &pb.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (api *API) listenConfig() {
	defer func() {
		if r := recover(); r != nil {
			api.Log.WithFields(logrus.Fields{
				"func":  "listenConfig",
				"event": "panic",
			}).Errorf("%s", r)
			api.Config.Ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", api.Config.Addr.Host+":"+fmt.Sprint(api.Config.Addr.Port))
	for err != nil {
		api.Log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(api.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", api.Config.Addr.Host+":"+fmt.Sprint(api.Config.Addr.Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, api)
	if gs.Serve(ln); err != nil {
		api.Log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}

func (api *API) listenData() {
	defer func() {
		if r := recover(); r != nil {
			api.Log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": "panic",
			}).Errorf("%s", r)
			api.Data.Ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", api.Data.Server.Host+":"+fmt.Sprint(api.Data.Server.Port))
	for err != nil {
		api.Log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(api.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", api.Data.Server.Host+":"+fmt.Sprint(api.Data.Server.Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, api)
	if gs.Serve(ln); err != nil {
		api.Log.WithFields(logrus.Fields{
			"func": "listenConfig",
		}).Fatalf("failed to serve: %s", err)
	}
}
