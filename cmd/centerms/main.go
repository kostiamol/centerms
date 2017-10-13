package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/services"
	"github.com/giperboloid/centerms/storages/redis"
)

func main() {
	var (
		st        = &storages.RedisStorage{}
		ctrl      = entities.ServicesController{StopChan: make(chan struct{})}
		reconnect = time.NewTicker(time.Second * 3)
	)
	st.SetServer(&StorageServer)

	ws := services.NewWebServer(entities.Server{Host: localhost, Port: webPort}, st, ctrl, logrus.New())
	go ws.Run()

	ss := services.NewStreamServer(entities.Server{Host: localhost, Port: streamPort}, st, ctrl, logrus.New())
	go ss.Run()

	dds := services.NewDevDataService(entities.Server{Host: localhost, Port: devDataPort}, st, ctrl, logrus.New(),
		reconnect)
	go dds.Run()

	dcs := services.NewDevConfigServer(entities.Server{Host: localhost, Port: devConfigPort}, st, ctrl, logrus.New(),
		reconnect, make(chan []string))
	go dcs.Run()

	ctrl.Wait()
	logrus.Info("centerms is down")
}
