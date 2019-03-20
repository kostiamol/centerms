package main

import (
	log_ "log"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/joho/godotenv"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
)

// todo: look through the handlers
// todo: retry
// todo: add Prometheus
// todo: update README.md
// todo: swagger

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log_.Fatalf("Load() failed: %s", err)
	}
}

func main() {
	devCfgChan := "dev_cfg"
	devDataChan := "dev_data"

	conf, err := cfg.NewConfig()
	if err != nil {
		log_.Fatalf("NewConfig(): %s", err)
	}

	log, err := cfg.NewLog(conf.Service.AppID, conf.Service.LogLevel)
	if err != nil {
		log_.Fatalf("NewLog(): %s", err)
	}

	redis, err := store.NewRedis(store.Addr{Host: conf.Store.Host, Port: int32(conf.Store.Port)}, conf.Store.Password)
	if err != nil {
		log.WithFields(logrus.Fields{
			"func":  "main",
			"event": cfg.EventStoreInit,
		}).Fatalf("NewRedis(): %s", err)
		os.Exit(1)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	// service initializations
	dataSvc := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			PubChan: devDataChan,
		})
	go dataSvc.Run()

	streamSvc := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logrus.NewEntry(log),
			Ctrl:       ctrl,
			Subscriber: redis,
			PubChan:    devDataChan,
			PortWS:     int32(conf.Service.PortWebSocket),
		})
	go streamSvc.Run()

	confSvc := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			SubChan: devCfgChan,
			Retry:   time.Duration(conf.Service.RetryTimeout),
		})
	go confSvc.Run()

	apiSvc := api.NewAPI(
		&api.APICfg{
			Log:          logrus.NewEntry(log),
			PubChan:      devCfgChan,
			PortRPC:      int32(conf.Service.PortRPC),
			PortREST:     int32(conf.Service.PortREST),
			CfgProvider:  confSvc,
			DataProvider: dataSvc,
			Retry:        time.Duration(conf.Service.RetryTimeout),
		})
	go apiSvc.Run()

	ctrl.Wait()

	log.WithFields(logrus.Fields{
		"func":  "main",
		"event": cfg.EventMSTerminated,
	}).Infof("%s is down", conf.Service.AppID)
}
