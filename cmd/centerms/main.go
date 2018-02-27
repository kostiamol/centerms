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

// todo: substitute req with common struct
// todo: fix logging in main package
// todo: reconnect
// todo: fill encapsulation gaps in structs

func main() {
	flag.Parse()
	checkCLIArgs()

	storageServer := entities.Address{
		Host: *storageHost,
		Port: *storagePort,
	}
	storage := storages.NewRedisStorage(storageServer, "redis", *ttl, *retry, logrus.NewEntry(log))
	if err := storage.Init(); err != nil {
		logrus.Error(err)
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
		entities.DevConfigSubject,
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
		entities.DevDataSubject,
	)
	go data.Run()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: config,
		DataService:   data,
		RetryInterval: *retry,
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
		entities.DevDataSubject,
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
		entities.DevConfigSubject,
	)
	go web.Run()

	ctrl.Wait()
	logrus.Info("center is down")
}
