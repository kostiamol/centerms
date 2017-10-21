package grpcsvc

import (
	"time"

	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/services"
	"net"
	"fmt"
	"google.golang.org/grpc"
	"github.com/giperboloid/centerms/pb"
	"golang.org/x/net/context"
)

type GRPCConfig struct {
	ConfigService services.ConfigService
	DataService   services.DataService
	Reconnect     *time.Ticker
	Server        entities.Server
}

func Init(c GRPCConfig) {
	s := newCenterServiceGRPC(c)
	go s.listenConfig()
	go s.listenData()
}

func newCenterServiceGRPC(c GRPCConfig) *API {
	return &API{
		Config:    c.ConfigService,
		Data:      c.DataService,
		Reconnect: c.Reconnect,
		Server:    c.Server,
	}
}

type API struct {
	Config    services.ConfigService
	Data      services.DataService
	Reconnect *time.Ticker
	Server    entities.Server
}

func (a *API) SetDevInitConfig(ctx context.Context, r *pb.SetDevInitConfigRequest) (*pb.SetDevInitConfigResponse, error) {
	dm := entities.DevMeta{
		Type: r.Meta.Type,
		Name: r.Meta.Name,
		MAC:  r.Meta.Mac,
	}

	dc,_ := a.Config.SetDevInitConfig(&dm)

	return &pb.SetDevInitConfigResponse{
		Config: dc.Data,
	}, nil
}

func (a *API) SaveDevData(ctx context.Context, r *pb.SaveDevDataRequest) (*pb.SaveDevDataResponse, error) {
	req := entities.Request{
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
	ln, err := net.Listen("tcp", a.Config.Server.Host+":"+fmt.Sprint(a.Config.Server.Port))
	if err != nil {
		a.Config.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range a.Reconnect.C {
			ln, err = net.Listen("tcp", a.Config.Server.Host+":"+fmt.Sprint(a.Config.Server.Port))
			if err != nil {
				a.Config.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
			}
		}
		a.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	gs.Serve(ln)
}

func (a *API) listenData() {
	ln, err := net.Listen("tcp", a.Data.Server.Host+":"+fmt.Sprint(a.Data.Server.Port))
	if err != nil {
		a.Data.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range a.Reconnect.C {
			ln, err = net.Listen("tcp", a.Data.Server.Host+":"+fmt.Sprint(a.Data.Server.Port))
			if err != nil {
				a.Data.Log.Errorf("DataService: Run(): Listen() has failed: %s", err)
			}
		}
		a.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterCenterServiceServer(gs, a)
	gs.Serve(ln)
}