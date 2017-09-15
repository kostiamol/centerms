package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/servers"
	"github.com/giperboloid/centerms/storages/redis"
)

func main() {
	var (
		st        = &storages.RedisDevStorage{}
		ctrl      = entities.ServersController{StopChan: make(chan struct{})}
		reconnect = time.NewTicker(time.Second * 1)
	)
	st.SetServer(&StorageServer)

	ws := servers.NewWebServer(entities.Server{Host: localhost, Port: webPort}, st, ctrl, logrus.New())
	go ws.Run()

	ss := servers.NewStreamServer(entities.Server{Host: localhost, Port: streamPort}, st, ctrl, logrus.New())
	go ss.Run()

	dds := servers.NewDevDataServer(entities.Server{Host: localhost, Port: devDataPort}, st, ctrl, logrus.New(),
		reconnect)
	go dds.Run()

	dcs := servers.NewDevConfigServer(entities.Server{Host: localhost, Port: devConfigPort}, st, ctrl, logrus.New(),
		reconnect, make(chan []string))
	go dcs.Run()

	ctrl.Wait()
	logrus.Info("All the servers are shut down")
}
