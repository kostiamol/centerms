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

type envVars struct {
	AppID         string `env:"APP_ID" envDefault:"centerms"`
	TTL           int    `env:"TTL" envDefault:"4"`
	Retry         int    `env:"RETRY" envDefault:"10"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	StoreHost     string `env:"STORE_HOST" envDefault:"127.0.0.1"`
	StorePort     int    `env:"STORE_PORT" envDefault:"6379"`
	StorePassword string `env:"STORE_PASSWORD" envDefault:"password"`
	PortRPC       int    `env:"RPC_PORT" envDefault:"8090"`
	PortREST      int    `env:"REST_PORT" envDefault:"8080"`
	PortWebSocket int    `env:"WEBSOCKET_PORT" envDefault:"8070"`
}

func main() {
	devCfgChan := "dev_cfg"
	devDataChan := "dev_data"

	log := &logrus.Logger{
		Out:       os.Stdout,
		Level:     logrus.DebugLevel,
		Formatter: new(logrus.TextFormatter),
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

	redis, err := store.NewRedis(store.Addr{Host: vars.StoreHost, Port: int32(vars.StorePort)}, vars.StorePassword)
	if err != nil {
		log.WithFields(logrus.Fields{
			"func":  "main",
			"event": cfg.EventStoreInit,
		}).Fatalf("NewRedis(): %s", err)
		os.Exit(1)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			PubChan: devDataChan,
		})
	go data.Run()

	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logrus.NewEntry(log),
			Ctrl:       ctrl,
			Subscriber: redis,
			PubChan:    devDataChan,
			PortWS:     int32(vars.PortWebSocket),
		})
	go stream.Run()

	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			SubChan: devCfgChan,
			Retry:   time.Duration(vars.TTL),
		})
	go conf.Run()

	api_ := api.NewAPI(
		&api.APICfg{
			Log:          logrus.NewEntry(log),
			PubChan:      devCfgChan,
			PortRPC:      int32(vars.PortRPC),
			PortREST:     int32(vars.PortREST),
			CfgProvider:  conf,
			DataProvider: data,
			Retry:        time.Duration(vars.Retry),
		})
	go api_.Run()

	ctrl.Wait()

	log.WithFields(logrus.Fields{
		"func":  "main",
		"event": cfg.EventMSTerminated,
	}).Infof("%s is down", vars.AppID)
}
