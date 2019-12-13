package api

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/store/model"

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
			a.log.With("event", log.EventPanic).Errorf("func serveRPC: %s", r)
		}
	}()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	l, err := net.Listen("tcp", ":"+fmt.Sprint(a.portRPC))
	if err != nil {
		a.log.Fatalf("func Listen: %s", err)
	}

	proto.RegisterCenterServiceServer(s, a)

	if err := s.Serve(l); err != nil {
		a.log.Fatalf("func Serve: %s", err)
	}
}

// Init returns device configuration. If the device hasn't been registered before, Init initializes
// configuration with default values that depend on the devices' type.
func (a *api) GetInitCfg(ctx context.Context, r *proto.GetInitCfgRequest) (*proto.GetInitCfgResponse, error) {
	m := &model.Meta{
		Type:  model.Type(r.Type),
		DevID: r.DevId,
	}

	c, err := a.cfgProvider.GetInitCfg(m)
	if err != nil {
		return nil, fmt.Errorf("func GetInitCfg: %s", err)
	}

	return &proto.GetInitCfgResponse{Cfg: c.Data}, nil
}

// SaveDevData saves device data.
func (a *api) SaveData(ctx context.Context, r *proto.SaveDataRequest) (*proto.SaveDataResponse, error) {
	d := model.Data{
		Time: r.Time,
		Meta: model.Meta{
			Type:  model.Type(r.Type),
			DevID: r.DevId,
		},
		Data: r.Data,
	}

	if err := a.dataProvider.SaveData(&d); err != nil {
		return nil, fmt.Errorf("func SaveData: %s", err)
	}

	return &proto.SaveDataResponse{Status: proto.Status_OK}, nil
}
