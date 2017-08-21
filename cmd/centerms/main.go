package main

import (
	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/servers"
)

func main() {

	dbClient := &db.RedisClient{}
	dbClient.SetDBServer(dbServer)

	controller := entities.RoutinesController{make(chan struct{})}

	httpServer := servers.NewHTTPServer(entities.Server{Host: localhost, Port: httpPort}, controller, dbClient)
	go httpServer.Run()

	wsServer := servers.NewWebSocketServer(entities.Server{Host: localhost, Port: wsPort}, controller, dbClient)
	go wsServer.Run()

	tcpDevConfigServer := servers.NewTCPDevConfigServerDefault(entities.Server{Host: localhost, Port: tcpConfigPort},
		controller, dbClient)
	go tcpDevConfigServer.Run()

	tcpDevDataServer := servers.NewTCPDevDataServerDefault(entities.Server{Host: localhost, Port: tcpDataPort},
		controller, dbClient)
	go tcpDevDataServer.Run()

	controller.Wait()
}
