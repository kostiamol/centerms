package main

import (
	"time"

	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/servers"
	"github.com/giperboloid/centerms/entities"
)

func main() {

	dbClient := &db.RedisClient{}
	dbClient.SetDBServer(dbServer)

	controller := entities.RoutinesController{make(chan struct{})}

	httpServer := servers.NewHTTPServer(entities.Server{Host: localhost, Port: httpPort}, controller, dbClient)
	go httpServer.Run()

	webSocketServer := servers.NewWebSocketServer(entities.Server{Host: localhost, Port: wsPort}, controller,dbClient)
	go webSocketServer.Run()

	tcpDevConfigServer := servers.NewTCPDevConfigServerDefault(entities.Server{Host: localhost, Port: tcpConfigPort},
		 controller, dbClient)
	go tcpDevConfigServer.Run()

	reconnect := time.NewTicker(time.Second * 1)
	tcpDevDataServer := servers.NewTCPDevDataServer(entities.Server{Host: localhost, Port: tcpDataPort},
		 reconnect, controller,dbClient)
	go tcpDevDataServer.Run()

	controller.Wait()
}
