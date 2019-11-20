package main

import (
	"github.com/kostiamol/centerms/event/pub"
	"github.com/kostiamol/centerms/metric"
	"github.com/kostiamol/centerms/trace"

	"github.com/joho/godotenv"
	"github.com/kostiamol/centerms/api"
	"github.com/kostiamol/centerms/cfg"
	"github.com/kostiamol/centerms/log"
	"github.com/kostiamol/centerms/store"
	"github.com/kostiamol/centerms/svc"
)

// todo: add proto validation
// todo: add fast loger
// todo: look through the handlers
// todo: check architecture (redis, websocket, nats, raw json data)
// todo: overall test coverage > 80%
// todo: check travis + badges
// todo: swagger
// todo: update README.md: helm (deployment, redis, nats), schema (arrows, k8s)
// todo: update docker-compose and test with fridgems
// todo: start fridgems refactoring

// runner is a contract for all the components (api + services).
type runner interface {
	Run()
}

func main() {
	loadCfgErr := godotenv.Load(".env")
	config, newCfgErr := cfg.New()
	logger := log.New(config.Service.AppID, config.Service.LogLevel)
	if loadCfgErr != nil {
		logger.Infof("func godotenv.Load: %s", loadCfgErr)
	}
	if newCfgErr != nil {
		logger.Fatalf("func cfg.New: %s", newCfgErr)
	}

	if err := trace.Init(config.Service.AppID, config.TraceAgent); err != nil {
		logger.Fatalf("func trace.Init: %s", err)
	}

	storer, err := store.New(
		&store.Cfg{
			Addr:             cfg.Addr{Host: config.Store.Addr.Host, Port: config.Store.Addr.Port},
			Password:         config.Store.Password,
			MaxIdlePoolConns: config.Store.MaxIdlePoolConns,
			IdleTimeout:      config.Store.IdleTimeout,
		})
	if err != nil {
		logger.Fatalf("func store.New: %s", err)
	}
	logger.With("event", log.EventStoreInit)

	publisher := pub.New(
		&pub.Cfg{
			Addr:          cfg.Addr{Host: config.Publisher.Addr.Host, Port: config.Publisher.Addr.Port},
			RetryTimeout:  config.Service.RetryTimeout,
			RetryAttempts: config.Service.RetryAttempts,
		})

	ctrl := svc.Ctrl{StopChan: make(chan struct{})}
	mtrc := metric.New(config.Service.AppID)
	dataChan := make(chan *svc.DevData)
	confChan := make(chan *svc.DevCfg)

	data := svc.NewDataService(
		&svc.DataServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			Metric:  mtrc,
			Store:   storer,
			PubChan: dataChan,
		})
	conf := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Log:       logger,
			Ctrl:      ctrl,
			Metric:    mtrc,
			Store:     storer,
			SubChan:   confChan,
			Publisher: publisher,
		})
	stream := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Log:     logger,
			Ctrl:    ctrl,
			Metric:  mtrc,
			SubChan: dataChan,
			PortWS:  config.Service.PortWebSocket,
		})
	a := api.New(
		&api.Cfg{
			Log:          logger,
			Ctrl:         ctrl,
			Metric:       mtrc,
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

	logger.With("event", log.EventMSShutdown)
	_ = logger.Flush()
}
