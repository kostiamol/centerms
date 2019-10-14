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
// todo: extract pub/sub from store into event package
// todo: deploy to minikube using helm chart
// todo: add Prometheus
// todo: update README.md
// todo: swagger

// runner is a contract for all the components (api + services).
type runner interface {
	Run()
}

func main() {
	var loadCfgErr, newCfgErr error
	if loadCfgErr = godotenv.Load(".env"); loadCfgErr != nil {
		loadCfgErr = fmt.Errorf("godotenv.Load(): %s", loadCfgErr)
	}

	config, newCfgErr := cfg.New()
	if newCfgErr != nil {
		newCfgErr = fmt.Errorf("cfg.New(): %s", newCfgErr)
	}

	logger := log.New(config.Service.AppID, config.Service.LogLevel)

	if loadCfgErr != nil || newCfgErr != nil {
		logger.Fatal(newCfgErr)
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
			Log:       logger,
			Ctrl:      ctrl,
			Store:     storer,
			Publisher: storer,
			PubChan:   cfg.DevDataChan,
		})
	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:           logger,
			Ctrl:          ctrl,
			Store:         storer,
			Subscriber:    storer,
			SubChan:       cfg.DevCfgChan,
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
	a := api.New(
		&api.Cfg{
			Log:          logger,
			PubChan:      cfg.DevCfgChan,
			PortRPC:      int32(config.Service.PortRPC),
			PortREST:     int32(config.Service.PortREST),
			Publisher:    storer,
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
