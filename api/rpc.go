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
	GetAddr() entity.Addr
}

type dataProvider interface {
	SaveDevData(data *entity.DevData)
	GetAddr() entity.Addr
}

type api_ struct {
	cfg   cfgProvider
	data  dataProvider
	retry time.Duration
	log   *logrus.Entry
}

// New creates and initializes a new instance of API service.
func New(c cfgProvider, d dataProvider, retry time.Duration, l *logrus.Entry) *api_ {
	return &api_{
		cfg:   c,
		data:  d,
		retry: retry,
		log:   l.WithFields(logrus.Fields{"svc": "api"}),
	}
}

// Run launches the goroutines for listening device data and configurations.
func (a *api_) Run() {
	go a.listenCfg()
	go a.listenData()
}

func (a *api_) listenCfg() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenCfg",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
		}
	}()

	l, err := net.Listen("tcp", a.cfg.GetAddr().Host+":"+fmt.Sprint(a.cfg.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenCfg",
		}).Errorf("Listen() has failed: %s", err)
		d := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*d + 1)
		l, err = net.Listen("tcp", a.cfg.GetAddr().Host+":"+fmt.Sprint(a.cfg.GetAddr().Port))
	}

	s := grpc.NewServer()
	proto.RegisterCenterServiceServer(s, a)
	if s.Serve(l); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenCfg",
		}).Fatalf("failed to serve: %s", err)
	}
}

func (a *api_) listenData() {
	defer func() {
		if r := recover(); r != nil {
			a.log.WithFields(logrus.Fields{
				"func":  "listenData",
				"event": entity.EventPanic,
			}).Errorf("%s", r)
		}
	}()

	l, err := net.Listen("tcp", a.data.GetAddr().Host+":"+fmt.Sprint(a.data.GetAddr().Port))
	for err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenData",
		}).Errorf("Listen() has failed: %s", err)
		d := time.Duration(rand.Intn(int(a.retry.Seconds())))
		time.Sleep(time.Second*d + 1)
		l, err = net.Listen("tcp", a.data.GetAddr().Host+":"+fmt.Sprint(a.data.GetAddr().Port))
	}

	s := grpc.NewServer()
	proto.RegisterCenterServiceServer(s, a)
	if s.Serve(l); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "listenCfg",
		}).Fatalf("failed to serve: %s", err)
	}
}

// SetDevInitConf sets device's initial configuration when it connects to the center for the first time using Cfg
// and returns that configuration to the device.
func (a *api_) SetDevInitCfg(ctx context.Context, req *proto.SetDevInitCfgRequest) (*proto.SetDevInitCfgResponse,
	error) {
	m := entity.DevMeta{
		Type: req.Meta.Type,
		Name: req.Meta.Name,
		MAC:  req.Meta.Mac,
	}

	c, err := a.cfg.SetDevInitCfg(&m)
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "SetDevInitCfg",
		}).Errorf("failed to set initial cfg: %s", err)
	}

	return &proto.SetDevInitCfgResponse{
		Cfg: c.Data,
	}, nil
}

// SaveDevData saves data from device using Data.
func (a *api_) SaveDevData(ctx context.Context, req *proto.SaveDevDataRequest) (*proto.SaveDevDataResponse, error) {
	d := entity.DevData{
		Time: req.Time,
		Meta: entity.DevMeta{
			Type: req.Meta.Type,
			Name: req.Meta.Name,
			MAC:  req.Meta.Mac,
		},
		Data: req.Data,
	}

	a.data.SaveDevData(&d)

	return &proto.SaveDevDataResponse{
		Status: "OK",
	}, nil
}
