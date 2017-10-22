package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/services"
	"github.com/giperboloid/centerms/storages/redis"
	"github.com/giperboloid/centerms/api/grpcsvc"
)

func main() {
	var (
		st        = &storages.RedisStorage{}
		ctrl      = entities.ServicesController{StopChan: make(chan struct{})}
		reconnect = time.NewTicker(time.Second * 3)
	)
	st.SetServer(&StorageServer)

	cs := services.NewConfigService(
		entities.Server{
			Host: localhost,
			Port: devConfigPort,
		},
		st,
		ctrl,
		logrus.New(),
		reconnect,
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
		reconnect,
	)
	go ds.Run()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: cs,
		DataService: ds,
		Reconnect: time.NewTicker(time.Second * 3),
	})

	ss := services.NewStreamService(
		entities.Server{
			Host: localhost,
			Port: streamPort,
		},
		st,
		ctrl,
		logrus.New(),
	)
	go ss.Run()

	ws := services.NewWebService(
		entities.Server{
			Host: localhost,
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
