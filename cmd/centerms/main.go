package main

import (
	"os"

	"github.com/kostiamol/centerms/params"

	"github.com/kostiamol/centerms/api"

	"flag"

	"github.com/Sirupsen/logrus"
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
			"event": params.EventStoreInit,
		}).Errorf("%s", err)
		os.Exit(1)
	}

	cfg := svc.NewCfgService(
		svc.Addr{
			Host: host,
			Port: *devCfgPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		*retry,
		svc.DevCfgChan,
		cfgAgentName,
		*ttl,
	)
	go cfg.Run()

	data := svc.NewDataService(
		svc.Addr{
			Host: host,
			Port: *devDataPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		svc.DevDataChan,
		dataAgentName,
		*ttl,
	)
	go data.Run()

	stream := svc.NewStreamService(
		svc.Addr{
			Host: host,
			Port: *streamPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		svc.DevDataChan,
		streamAgentName,
		*ttl,
	)
	go stream.Run()

	a := api.NewAPI(
		host,
		*rpcPort,
		*restPort,
		cfg,
		logrus.NewEntry(log),
		*retry,
		svc.DevCfgChan,
		webAgentName,
		*ttl,
	)
	go a.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": params.EventMSTerminated,
	}).Info("centerms is down")
}
