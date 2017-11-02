package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/api/grpcsvc"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/services"
	"github.com/giperboloid/centerms/storages/redis"
)

func main() {
	var (
		st            = &storages.RedisStorage{RetryInterval: retryInterval}
		ctrl          = entities.ServicesController{StopChan: make(chan struct{})}
		storageServer = entities.Server{
			Host: storageHost,
			Port: storagePort,
		}
	)
	st.SetServer(storageServer)

	cs := services.NewConfigService(
		entities.Server{
			Host: localhost,
			Port: devConfigPort,
		},
		st,
		ctrl,
		logrus.New(),
	)
	go cs.Run()

	ds := services.NewDataService(
		entities.Server{
			Host: localhost,
			Port: devDataPort,
		},
		st,
		ctrl,
		logrus.New(),
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
		st,
		ctrl,
		logrus.New(),
	)
	go ss.Run()

	ws := services.NewWebService(
		entities.Server{
			Host: webHost,
			Port: webPort,
		},
		st,
		ctrl,
		logrus.New(),
	)
	go ws.Run()

	ctrl.Wait()
	logrus.Info("center is down")
}
