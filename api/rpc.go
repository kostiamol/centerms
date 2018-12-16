package api

import (
	"math/rand"
	"time"

	"github.com/kostiamol/centerms/proto"

	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entity"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type cfgProvider interface {
	SetDevInitCfg(m *entity.DevMeta) (*entity.DevCfg, error)
}

func (a *API) runRPC() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenCfg",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
		}
	}()

	l, err := net.Listen("tcp", a.host+":"+fmt.Sprint(a.rpcPort))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "runRPC",
		}).Errorf("Listen() has failed: %s", err)
		d := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*d + 1)
		l, err = net.Listen("tcp", a.host+":"+fmt.Sprint(a.rpcPort))
	}

	s := grpc.NewServer()
	proto.RegisterCenterServiceServer(s, a)
	if err := s.Serve(l); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "runRPC",
		}).Fatalf("failed to serve: %s", err)
	}
}

// SetDevInitCfg sets device's initial configuration when it connects to the center for the first time using Cfg
// and returns that configuration to the device.
func (a *API) SetDevInitCfg(ctx context.Context, req *proto.SetDevInitCfgRequest) (*proto.SetDevInitCfgResponse,
	error) {
	conn, err := a.store.CreateConn()
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	m := entity.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	c, err := a.cfgProvider.SetDevInitCfg(&m)
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("failed to set init cfg: %s", err)
	}

	return &proto.SetDevInitCfgResponse{
		Cfg: c.Data,
	}, nil
}

// SaveDevData saves data from device using Data.
func (a *API) SaveDevData(ctx context.Context, req *proto.SaveDevDataRequest) (*proto.SaveDevDataResponse, error) {
	conn, err := a.store.CreateConn()
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("%s", err)
		return nil, err
	}
	defer conn.CloseConn()

	d := entity.DevData{
		Time: req.Time,
		Meta: entity.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	if err := conn.SaveDevData(&d); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "SaveDevData",
		}).Errorf("failed to set init cfg: %s", err)
	}

	return &proto.SaveDevDataResponse{
		Status: "OK",
	}, nil
}
