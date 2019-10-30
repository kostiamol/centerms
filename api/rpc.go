package api

import (
	"github.com/kostiamol/centerms/svc"

	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/kostiamol/centerms/cfg"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/kostiamol/centerms/proto"

	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func (a *api) serveRPC() {
	defer func() {
		if r := recover(); r != nil {
			a.log.With("event", cfg.EventPanic).Errorf("serveRPC(): %s", r)
		}
	}()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	l, err := net.Listen("tcp", ":"+fmt.Sprint(a.portRPC))
	if err != nil {
		a.log.Fatalf("Listen(): %s", err)
	}

	proto.RegisterCenterServiceServer(s, a)

	if err := s.Serve(l); err != nil {
		a.log.Fatalf("Serve(): %s", err)
	}
}

// SetDevInitCfg sets device's initial configuration when it connects to the center for the first
// time using CfgProvider and returns that configuration to the device.
func (a *api) SetDevInitCfg(ctx context.Context, r *proto.SetDevInitCfgRequest) (*proto.SetDevInitCfgResponse, error) {
	m := svc.DevMeta{
		Type: r.Meta.Type,
		Name: r.Meta.Name,
		MAC:  r.Meta.Mac,
	}

	c, err := a.cfgProvider.SetDevInitCfg(&m)
	if err != nil {
		return nil, fmt.Errorf("SetDevInitCfg(): %s", err)
	}

	return &proto.SetDevInitCfgResponse{Cfg: c.Data}, nil
}

// SaveDevData saves data from device using DataProvider.
func (a *api) SaveDevData(ctx context.Context, r *proto.SaveDevDataRequest) (*proto.SaveDevDataResponse, error) {
	d := svc.DevData{
		Time: r.Time,
		Meta: svc.DevMeta{
			Type: r.Meta.Type,
			Name: r.Meta.Name,
			MAC:  r.Meta.Mac,
		},
		Data: r.Data,
	}

	if err := a.dataProvider.SaveDevData(&d); err != nil {
		return nil, fmt.Errorf("SaveDevData(): %s", err)
	}

	return &proto.SaveDevDataResponse{Status: "OK"}, nil
}
