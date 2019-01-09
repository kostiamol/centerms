package main

import (
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/store"

	"github.com/kostiamol/centerms/api"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/svc"
)

// todo: add Prometheus
// todo: update README.md
// todo: reconnect + conn pool

type envVars struct {
	AppID         string `env:"APP_ID" envDefault:"centerms"`
	TTL           int    `env:"TTL" envDefault:"4"`
	Retry         int    `env:"RETRY" envDefault:"10"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	StoreHost     string `env:"STORE_HOST" envDefault:"127.0.0.1"`
	StorePort     int    `env:"STORE_PORT" envDefault:"6379"`
	RPCPort       int    `env:"RPC_PORT" envDefault:"8090"`
	RESTPort      int    `env:"REST_PORT" envDefault:"8080"`
	WebSocketPort int    `env:"WEBSOCKET_PORT" envDefault:"8070"`
}

func main() {
	devCfgChan := "dev_cfg"
	devDataChan := "dev_data"

	log := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Level:     logrus.DebugLevel,
	}

	vars := envVars{}
	if err := env.Parse(&vars); err != nil {
		log.WithFields(logrus.Fields{
			"func": "main",
		}).Fatalf("Parse() failed: %s", err)
		os.Exit(1)
	}

	lvl, err := logrus.ParseLevel(vars.LogLevel)
	if err != nil {
		log.WithFields(logrus.Fields{
			"func": "main",
		}).Fatalf("ParseLevel() failed: %s", err)
	} else {
		log.SetLevel(lvl)
	}

	redis := store.NewRedis(svc.Addr{Host: vars.StoreHost, Port: vars.StorePort})
	if err := redis.Init(); err != nil {
		log.WithFields(logrus.Fields{
			"func":  "main",
			"event": cfg.EventStoreInit,
		}).Fatalf("Init() failed: %s", err)
		os.Exit(1)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Store:   redis,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			PubChan: devDataChan,
		})
	go data.Run()

	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Port:    vars.WebSocketPort,
			Store:   redis,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			PubChan: devDataChan,
		})
	go stream.Run()

	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Store:   redis,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			Retry:   time.Duration(vars.TTL),
			SubChan: devCfgChan,
		})
	go conf.Run()

	a := api.NewAPI(
		&api.APICfg{
			RPCPort:     vars.RPCPort,
			RESTPort:    vars.RESTPort,
			CfgProvider: conf,
			Log:         logrus.NewEntry(log),
			PubChan:     devCfgChan,
		})
	go a.Run()

	agent := svc.NewMeshAgent(
		&svc.MeshAgentCfg{
			Name: vars.AppID,
			Port: vars.RESTPort,
			TTL:  time.Duration(vars.TTL),
			Log:  logrus.NewEntry(log),
		})
	go agent.Run()

	ctrl.Wait()

	log.WithFields(logrus.Fields{
		"func":  "main",
		"event": cfg.EventMSTerminated,
	}).Infof("%s is down", vars.AppID)
}
