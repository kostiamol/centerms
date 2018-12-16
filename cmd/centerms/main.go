package main

import (
	"os"

	"github.com/kostiamol/centerms/api"

	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entity"
	"github.com/kostiamol/centerms/svc"
)

// todo: add Prometheus
// todo: update README.md
// todo: reconnect + conn pool

func main() {
	flag.Parse()

	if err := store.Init(); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "main",
			"event": entity.EventStoreInit,
		}).Errorf("%s", err)
		os.Exit(1)
	}

	cfg := svc.NewCfgService(
		entity.Addr{
			Host: host,
			Port: *devCfgPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		*retry,
		entity.DevCfgChan,
		cfgAgentName,
		*ttl,
	)
	go cfg.Run()

	data := svc.NewDataService(
		entity.Addr{
			Host: host,
			Port: *devDataPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		entity.DevDataChan,
		dataAgentName,
		*ttl,
	)
	go data.Run()

	stream := svc.NewStreamService(
		entity.Addr{
			Host: host,
			Port: *streamPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		entity.DevDataChan,
		streamAgentName,
		*ttl,
	)
	go stream.Run()

	a := api.NewAPI(
		host,
		*rpcPort,
		*restPort,
		cfg,
		store,
		logrus.NewEntry(log),
		*retry,
		entity.DevCfgChan,
		webAgentName,
		*ttl,
	)
	go a.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": entity.EventMSTerminated,
	}).Info("centerms is down")
}
