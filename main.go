package main

import (
	"fmt"

	"github.com/joho/godotenv"
	api_ "github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
)

// todo: stream and other svc: subscribe error: separate into another routine
// todo: init cfg with pub chan
// todo: handle all errors
// todo: store/meta.go interface usage
// todo: look through the handlers
// todo: add Prometheus
// todo: update README.md
// todo: swagger

// runner is a contract for all the components (api + services).
type runner interface {
	Run()
}

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
		&store.Cfg{
			Addr:             cfg.Addr{Host: config.Store.Addr.Host, Port: config.Store.Addr.Port},
			Password:         config.Store.Password,
			MaxIdlePoolConns: config.Store.MaxIdlePoolConns,
			IdleTimeout:      config.Store.IdleTimeout,
		})
	if err != nil {
		logger.Fatalf("store.New(): %s", err)
	}

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			Store:   storer,
			PubChan: cfg.DevDataChan,
		})
	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:           logger,
			Ctrl:          ctrl,
			Store:         storer,
			Subscriber:    storer,
			SubChan:       cfg.DevCfgChan,
			Publisher:     storer,
			RetryTimeout:  config.Service.RetryTimeout,
			RetryAttempts: config.Service.RetryAttempts,
			NATSAddr:      cfg.Addr{Host: config.NATS.Addr.Host, Port: config.NATS.Addr.Port},
		})
	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:        logger,
			Ctrl:       ctrl,
			Subscriber: storer,
			SubChan:    cfg.DevDataChan,
			PortWS:     config.Service.PortWebSocket,
		})
	api := api_.New(
		&api_.Cfg{
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

	components := []runner{data, conf, stream, api}
	for _, c := range components {
		go c.Run()
	}

	ctrl.Wait(config.Service.RoutineTerminationTimeout)

	logger.Info("service is down")
}
