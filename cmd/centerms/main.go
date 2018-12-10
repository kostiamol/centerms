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

	c := svc.NewCfg(
		entity.Addr{
			Host: localhost,
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
	go c.Run()

	d := svc.NewData(
		entity.Addr{
			Host: localhost,
			Port: *devDataPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		entity.DevDataChan,
		dataAgentName,
		*ttl,
	)
	go d.Run()

	a := api.New(
		c,
		d,
		*retry,
		logrus.NewEntry(log),
	)
	a.Run()

	s := svc.NewStream(
		entity.Addr{
			Host: webHost,
			Port: *streamPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		entity.DevDataChan,
		streamAgentName,
		*ttl,
	)
	go s.Run()

	w := svc.NewWeb(
		entity.Addr{
			Host: webHost,
			Port: *webPort,
		},
		store,
		ctrl,
		logrus.NewEntry(log),
		entity.DevCfgChan,
		webAgentName,
		*ttl,
	)
	go w.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": entity.EventMSTerminated,
	}).Info("center is down")
}
