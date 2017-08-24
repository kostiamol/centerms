package main

import (
	"time"

	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/servers"
	"github.com/giperboloid/centerms/storages/redis"
)

func main() {
	var (
		storage    = &storages.RedisStorage{}
		reconnect  = time.NewTicker(time.Second * 1)
		ctrl = entities.RoutinesController{StopChan: make(chan struct{})}
	)

	storage.SetServer(&StorageServer)

	ws := servers.NewWebServer(entities.Server{Host: localhost, Port: webPort}, ctrl, storage)
	go ws.Run()

	ss := servers.NewStreamServer(entities.Server{Host: localhost, Port: streamPort}, ctrl, storage)
	go ss.Run()

	dds := servers.NewDevDataServer(entities.Server{Host: localhost, Port: devDataPort}, reconnect,
		ctrl, storage)
	go dds.Run()

	dcs := servers.NewDevConfigServer(entities.Server{Host: localhost, Port: devConfigPort}, reconnect,
		ctrl, storage, make(chan []string), make(chan struct{}))
	go dcs.Run()

	ctrl.Wait()
}
