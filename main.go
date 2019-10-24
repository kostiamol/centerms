package main

import (
	"github.com/kostiamol/centerms/event/pub"

	"github.com/joho/godotenv"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
)

// todo: look through the handlers
// todo: deploy to minikube using helm chart
// todo: add Prometheus
// todo: update README.md
// todo: swagger

// runner is a contract for all the components (api + services).
type runner interface {
	Run()
}

func main() {
	loadCfgErr := godotenv.Load(".env")
	config, newCfgErr := cfg.New()
	logger := log.New(config.Service.AppID, config.Service.LogLevel)
	if loadCfgErr != nil {
		logger.Infof("godotenv.Load(): %s", loadCfgErr)
	}
	if newCfgErr != nil {
		logger.Fatalf("cfg.New(): %s", newCfgErr)
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

	publisher := pub.New(
		&pub.Cfg{
			Addr:          cfg.Addr{Host: config.NATS.Addr.Host, Port: config.NATS.Addr.Port},
			RetryTimeout:  config.Service.RetryTimeout,
			RetryAttempts: config.Service.RetryAttempts,
		})

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}
	dataChan := make(chan *svc.DevData)
	confChan := make(chan *svc.DevCfg)

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			Store:   storer,
			PubChan: dataChan,
		})
	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:       logger,
			Ctrl:      ctrl,
			Store:     storer,
			SubChan:   confChan,
			Publisher: publisher,
		})
	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			SubChan: dataChan,
			PortWS:  config.Service.PortWebSocket,
		})
	a := api.New(
		&api.Cfg{
			Log:          logger,
			PubChan:      confChan,
			PortRPC:      config.Service.PortRPC,
			PortREST:     config.Service.PortREST,
			CfgProvider:  conf,
			DataProvider: data,
			Retry:        config.Service.RetryTimeout,
			PublicKey:    config.Token.PublicKey,
			PrivateKey:   config.Token.PrivateKey,
		})

	components := []runner{data, conf, stream, a}
	for _, c := range components {
		go c.Run()
	}

	ctrl.Wait(config.Service.RoutineTerminationTimeout)

	logger.Info("service is down")
	_ = logger.Flush()
}
