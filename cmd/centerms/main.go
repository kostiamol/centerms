package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/grpcsvc"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
	"github.com/kostiamol/centerms/storages/redis"
)

func main() {
	var (
		log = &logrus.Logger{
			Out:       os.Stdout,
			Formatter: new(logrus.TextFormatter),
			Level:     logrus.DebugLevel,
		}
		storage       = &storages.RedisStorage{RetryInterval: retryInterval}
		ctrl          = entities.ServiceController{StopChan: make(chan struct{})}
		storageServer = entities.Server{
			Host: storageHost,
			Port: storagePort,
		}
	)
	storage.SetServer(storageServer)

	cs := services.NewConfigService(
		entities.Server{
			Host: localhost,
			Port: devConfigPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		retryInterval,
		entities.DevConfigSubject,
	)
	go cs.Run()

	ds := services.NewDataService(
		entities.Server{
			Host: localhost,
			Port: devDataPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataSubject,
	)
	go ds.Run()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: cs,
		DataService:   ds,
		RetryInterval: retryInterval,
	})

	ss := services.NewStreamService(
		entities.Server{
			Host: webHost,
			Port: streamPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevDataSubject,
	)
	go ss.Run()

	ws := services.NewWebService(
		entities.Server{
			Host: webHost,
			Port: webPort,
		},
		storage,
		ctrl,
		logrus.NewEntry(log),
		entities.DevConfigSubject,
	)
	go ws.Run()

	ctrl.Wait()
	logrus.Info("center is down")
}
