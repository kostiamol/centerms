package rpc

import (
	"math/rand"
	"time"

	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entity"
	"github.com/kostiamol/centerms/svc"
	"github.com/kostiamol/centerms/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Cfg holds services for handling device data and configurations, retry interval and log.
type Cfg struct {
	Cfg   *svc.Cfg
	Data  *svc.Data
	Retry time.Duration
	Log   *logrus.Entry
}

// Init initializes gRPC and starts goroutines for listening device data and configs.
func Init(c Cfg) {
	s := newCenterServiceGRPC(c)
	go s.listenConf()
	go s.listenData()
}

func newCenterServiceGRPC(c Cfg) *service {
	return &service{
		conf:  *c.Cfg,
		data:  *c.Data,
		retry: c.Retry,
		log:   c.Log.WithFields(logrus.Fields{"service": "api"}),
	}
}

type service struct {
	conf  svc.Cfg
	data  svc.Data
	ctrl  svc.Ctrl
	retry time.Duration
	log   *logrus.Entry
}

// SetDevInitConf sets device's initial configuration when it connects to the center for the first time using Cfg
// and returns that configuration to the device.
func (a *service) SetDevInitConfig(ctx context.Context, req *api.SetDevInitConfigRequest) (*api.SetDevInitConfigResponse,
	error) {
	meta := entity.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	c, _ := a.conf.SetDevInitCfg(&meta)

	return &api.SetDevInitConfigResponse{
		Config: c.Data,
	}, nil
}

// SaveDevData saves data from device using Data.
func (a *service) SaveDevData(ctx context.Context, req *api.SaveDevDataRequest) (*api.SaveDevDataResponse, error) {
	data := entity.DevData{
		Time: req.Time,
		Meta: entity.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	a.data.SaveDevData(&data)

	return &api.SaveDevDataResponse{
		Status: "OK",
	}, nil
}

func (a *service) listenConf() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenConf",
				"event": entity.EventPanic,
			}).Errorf("%a", r)
			a.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.conf.GetAddr().Host+":"+fmt.Sprint(a.conf.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConf",
		}).Errorf("Listen() has failed: %a", err)
		duration := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.conf.GetAddr().Host+":"+fmt.Sprint(a.conf.GetAddr().Port))
	}

	gs := grpc.NewServer()
	api.RegisterCenterServiceServer(gs, a)
	if gs.Serve(ln); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConf",
		}).Fatalf("failed to serve: %a", err)
	}
}

func (a *service) listenData() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": entity.EventPanic,
			}).Errorf("%a", r)
			a.ctrl.StopChan <- struct{}{}
		}
	}()

	ln, err := net.Listen("tcp", a.data.GetAddr().Host+":"+fmt.Sprint(a.data.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %a", err)
		duration := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		ln, err = net.Listen("tcp", a.data.GetAddr().Host+":"+fmt.Sprint(a.data.GetAddr().Port))
	}

	gs := grpc.NewServer()
	api.RegisterCenterServiceServer(gs, a)
	if gs.Serve(ln); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenConf",
		}).Fatalf("failed to serve: %a", err)
	}
}
