package main

import (
	"github.com/joho/godotenv"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
	"github.com/sirupsen/logrus"
)

// todo: look through the handlers
// todo: retry
// todo: add Prometheus
// todo: update README.md
// todo: swagger

func init() {
	if err := godotenv.Load(".env"); err != nil {
		logrus.Errorf("Load(): %s", err)
	}
}

func main() {
	config, err := cfg.NewConfig()
	if err != nil {
		logrus.Fatalf("NewConfig(): %s", err)
	}

	store, err := store.New(store.Addr{Host: config.Store.Host, Port: config.Store.Port}, config.Store.Password)
	if err != nil {
		logrus.Fatalf("New(): %s", err)
	}

	log, err := cfg.NewLog(config.Service.AppID, config.Service.LogLevel)
	if err != nil {
		logrus.Fatalf("NewLog(): %s", err)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	// initialization of the services and api
	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   store,
			PubChan: cfg.DevDataChan,
		})
	go data.Run()

	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logrus.NewEntry(log),
			Ctrl:       ctrl,
			Subscriber: store,
			PubChan:    cfg.DevDataChan,
			PortWS:     int32(config.Service.PortWebSocket),
		})
	go stream.Run()

	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:     logrus.NewEntry(log),
			Ctrl:    ctrl,
			Store:   store,
			SubChan: cfg.DevCfgChan,
			Retry:   config.Service.RetryTimeout,
		})
	go conf.Run()

	api := api.New(
		&api.Cfg{
			Log:          logrus.NewEntry(log),
			PubChan:      cfg.DevCfgChan,
			PortRPC:      int32(config.Service.PortRPC),
			PortREST:     int32(config.Service.PortREST),
			CfgProvider:  conf,
			DataProvider: data,
			Retry:        config.Service.RetryTimeout,
			PublicKey:    config.Token.PublicKey,
			PrivateKey:   config.Token.PrivateKey,
		})
	go api.Run()

	ctrl.Wait()

	logrus.Infof("%s is down", config.Service.AppID)
}
