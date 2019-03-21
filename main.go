package main

import (
	log_ "log"
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
		log_.Fatalf("Load(): %s", err)
	}
}

func main() {
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
		log_.Fatalf("NewRedis(): %s", err)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	// initialization of the services
	dataSvc := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			PubChan: cfg.DevDataChan,
		})
	go dataSvc.Run()

	streamSvc := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logrus.NewEntry(log),
			Ctrl:       ctrl,
			Subscriber: redis,
			PubChan:    cfg.DevDataChan,
			PortWS:     int32(conf.Service.PortWebSocket),
		})
	go streamSvc.Run()

	confSvc := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   redis,
			SubChan: cfg.DevCfgChan,
			Retry:   time.Duration(conf.Service.RetryTimeout),
		})
	go confSvc.Run()

	apiSvc := api.NewAPI(
		&api.APICfg{
			Log:          logrus.NewEntry(log),
			PubChan:      cfg.DevCfgChan,
			PortRPC:      int32(conf.Service.PortRPC),
			PortREST:     int32(conf.Service.PortREST),
			CfgProvider:  confSvc,
			DataProvider: dataSvc,
			Retry:        time.Duration(conf.Service.RetryTimeout),
			PublicKey:    conf.Token.PublicKey,
			PrivateKey:   conf.Token.PrivateKey,
		})
	go apiSvc.Run()

	ctrl.Wait()

	log_.Printf("%s is down", conf.Service.AppID)
}
