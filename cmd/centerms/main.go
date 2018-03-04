package main

import (
	"flag"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/grpcsvc"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
	"github.com/kostiamol/centerms/storages/redis"
)

// todo: "extract" events
// todo: add comments and tidy up
// todo: substitute map[string][]string with []byte
// todo: add consul agent to center
// todo: reconnect

func main() {
	flag.Parse()
	checkCLIArgs()

	storageServer := entities.Address{
		Host: *storageHost,
		Port: *storagePort,
	}
	storage := storages.NewRedisStorage(storageServer, "redis", *ttl, *retry, logrus.NewEntry(log))
	if err := storage.Init(); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "main",
			"event": "storage_init",
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
	)
	go web.Run()

	ctrl.Wait()
	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": "ms_termination",
	}).Info("center is down")
}
