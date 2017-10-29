package grpcsvc

import (
	"math/rand"
	"time"

	"fmt"
	"net"

	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/pb"
	"github.com/giperboloid/centerms/services"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type GRPCConfig struct {
	ConfigService *services.ConfigService
	DataService   *services.DataService
	RetryInterval time.Duration
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
	}
}

type API struct {
	Config        services.ConfigService
	Data          services.DataService
	RetryInterval time.Duration
}

func (a *API) SetDevInitConfig(ctx context.Context, r *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	dm := entities.DevMeta{
		Type: r.Meta.Type,
		Name: r.Meta.Name,
		MAC:  r.Meta.Mac,
	}

	dc, _ := a.Config.SetDevInitConfig(&dm)

	return &pb.SetDevInitConfigResponse{
		Config: dc.Data,
	}, nil
}

func (a *API) SaveDevData(ctx context.Context, r *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	req := entities.SaveDevDataRequest{
		Time: r.Time,
		Meta: entities.DevMeta{
			Type: r.Meta.Type,
			Name: r.Meta.Name,
			MAC:  r.Meta.Mac,
		},
		Data: r.Data,
	}

	a.Data.SaveDevData(&req)

	return &pb.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (a *API) listenConfig() {
	defer func() {
		if r := recover(); r != nil {
			a.Config.Log.Errorf("API: listenConfig(): panic(): %s", r)
			a.Config.Controller.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.Config.Server.Host+":"+fmt.Sprint(a.Config.Server.Port))
	for err != nil {
		a.Config.Log.Errorf("API: listenConfig(): Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(a.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.Config.Server.Host+":"+fmt.Sprint(a.Config.Server.Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	gs.Serve(ln)
}

func (a *API) listenData() {
	defer func() {
		if r := recover(); r != nil {
			a.Data.Log.Errorf("API: listenData(): panic(): %s", r)
			a.Data.Controller.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.Data.Server.Host+":"+fmt.Sprint(a.Data.Server.Port))

	for err != nil {
		a.Config.Log.Errorf("API: listenData(): Listen() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(a.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.Data.Server.Host+":"+fmt.Sprint(a.Data.Server.Port))
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	gs.Serve(ln)
}
