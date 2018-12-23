package main

import (
	"os"

	"github.com/kostiamol/centerms/cfg"

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
			"event": cfg.EventStoreInit,
		}).Errorf("%s", err)
		os.Exit(1)
	}

	d := svc.NewDataService(
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
	go d.Run()

	s := svc.NewStreamService(
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
	go s.Run()

	c := svc.NewCfgService(
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
	go c.Run()

	a := api.NewAPI(
		host,
		*rpcPort,
		*restPort,
		c,
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
		"event": cfg.EventMSTerminated,
	}).Info("centerms is down")
}
