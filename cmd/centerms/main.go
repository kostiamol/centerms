package main

import (
	"github.com/kostiamol/centerms/api/rpc"
	"os"

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

	cfg := svc.NewCfg(
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
	go cfg.Run()

	data := svc.NewData(
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
	go data.Run()

	rpc.Init(rpc.Cfg{
		Cfg:   cfg,
		Data:  data,
		Retry: *retry,
		Log:   logrus.NewEntry(log),
	})

	stream := svc.NewStream(
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
	go stream.Run()

	web := svc.NewWeb(
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
	go web.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": entity.EventMSTerminated,
	}).Info("center is down")
}
