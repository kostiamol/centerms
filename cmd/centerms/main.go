package main

import (
	"os"

	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/grpcsvc"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
)

// todo: add Prometheus
// todo: update README.md
// todo: reconnect + conn pool

func main() {
	flag.Parse()

	if err := storage.Init(); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "main",
			"event": entities.EventStorageInit,
		}).Errorf("%s", err)
		os.Exit(1)
	}

	config := services.NewConfigService(
		entities.Address{
			Host: localhost,
			Port: *devConfigPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		*retry,
		entities.DevConfigChan,
		configAgentName,
		*ttl,
	)
	go config.Run()

	data := services.NewDataService(
		entities.Address{
			Host: localhost,
			Port: *devDataPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataChan,
		dataAgentName,
		*ttl,
	)
	go data.Run()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: config,
		DataService:   data,
		Retry:         *retry,
		Log:           logrus.NewEntry(log),
	})

	stream := services.NewStreamService(
		entities.Address{
			Host: webHost,
			Port: *streamPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataChan,
		streamAgentName,
		*ttl,
	)
	go stream.Run()

	web := services.NewWebService(
		entities.Address{
			Host: webHost,
			Port: *webPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevConfigChan,
		webAgentName,
		*ttl,
	)
	go web.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": entities.EventMSTerminated,
	}).Info("center is down")
}
