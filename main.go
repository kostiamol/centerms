package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
)

// todo: look through the handlers
// todo: retry
// todo: add Prometheus
// todo: update README.md
// todo: swagger
// todo: stream 15sec hardcoded

func main() {
	var loadCfgErr, initCfgErr error
	if loadCfgErr = godotenv.Load(".env"); loadCfgErr != nil {
		loadCfgErr = fmt.Errorf("Load(): %s", loadCfgErr)
	}

	config, initCfgErr := cfg.InitConfig()
	if initCfgErr != nil {
		initCfgErr = fmt.Errorf("InitConfig(): %s", initCfgErr)
	}

	logger := log.New(config.Service.AppID, config.Service.LogLevel)
	defer logger.Flush() // nolint

	if loadCfgErr != nil || initCfgErr != nil {
		logger.Fatal(initCfgErr)
	}

	storer, err := store.New(
		cfg.Addr{Host: config.Store.Addr.Host, Port: config.Store.Addr.Port},
		config.Store.Password)
	if err != nil {
		logger.Fatalf("New(): %s", err)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			Store:   storer,
			PubChan: cfg.DevDataChan,
		})
	go data.Run()

	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logger,
			Ctrl:       ctrl,
			Subscriber: storer,
			PubChan:    cfg.DevDataChan,
			PortWS:     int32(config.Service.PortWebSocket),
		})
	go stream.Run()

	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:      logger,
			Ctrl:     ctrl,
			Store:    storer,
			SubChan:  cfg.DevCfgChan,
			Retry:    config.Service.RetryTimeout,
			NATSAddr: cfg.Addr{Host: config.NATS.Addr.Host, Port: config.NATS.Addr.Port},
		})
	go conf.Run()

	a := api.New(
		&api.Cfg{
			Log:          logger,
			PubChan:      cfg.DevCfgChan,
			PortRPC:      int32(config.Service.PortRPC),
			PortREST:     int32(config.Service.PortREST),
			CfgProvider:  conf,
			DataProvider: data,
			Retry:        config.Service.RetryTimeout,
			PublicKey:    config.Token.PublicKey,
			PrivateKey:   config.Token.PrivateKey,
		})
	go a.Run()

	ctrl.Wait()

	logger.Infof("%s is down", config.Service.AppID)
}
