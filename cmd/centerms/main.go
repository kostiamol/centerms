package main

import (
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/servers"
	"github.com/giperboloid/centerms/storages/redis"
)

func main() {

	storage := &storages.RedisStorage{}
	storage.SetServer(StorageServer)

	controller := entities.RoutinesController{StopChan: make(chan struct{})}

	httpServer := servers.NewWebServer(entities.Server{Host: localhost, Port: httpPort}, controller, storage)
	go httpServer.Run()

	wsServer := servers.NewWebSocketServer(entities.Server{Host: localhost, Port: wsPort}, controller, storage)
	go wsServer.Run()

	tcpDevConfigServer := servers.NewDevConfigServerDefault(entities.Server{Host: localhost, Port: tcpConfigPort},
		controller, storage)
	go tcpDevConfigServer.Run()

	tcpDevDataServer := servers.NewDevDataServerDefault(entities.Server{Host: localhost, Port: tcpDataPort},
		controller, storage)
	go tcpDevDataServer.Run()

	controller.Wait()
}
