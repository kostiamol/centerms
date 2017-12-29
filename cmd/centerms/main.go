package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/api/grpcsvc"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
	"github.com/kostiamol/centerms/storages/redis"
)

func main() {
	var (
		st            = &storages.RedisStorage{RetryInterval: retryInterval}
		ctrl          = entities.ServiceController{StopChan: make(chan struct{})}
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
		retryInterval,
		entities.DevConfigSubject,
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
		st,
		ctrl,
		logrus.New(),
		entities.DevDataSubject,
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
		entities.DevConfigSubject,
	)
	go ws.Run()

	ctrl.Wait()
	logrus.Info("center is down")
}
