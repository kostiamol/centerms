package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type TCPDevDataServer struct {
	LocalServer entities.Server
	Reconnect   *time.Ticker
	Controller  entities.RoutinesController
	DbClient    db.Client
}

func NewTCPDevDataServer(local entities.Server, reconnect *time.Ticker, controller entities.RoutinesController, dbClient db.Client) *TCPDevDataServer {
	return &TCPDevDataServer{
		LocalServer: local,
		Reconnect:   reconnect,
		Controller:  controller,
		DbClient:    dbClient,
	}
}

func NewTCPDevDataServerDefault(local entities.Server, controller entities.RoutinesController, dbClient db.Client) *TCPDevDataServer {
	reconnect := time.NewTicker(time.Second * 1)
	return NewTCPDevDataServer(local, reconnect, controller, dbClient)
}

func (server *TCPDevDataServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			server.Controller.Close()
			errors.New("TCPDevDataServer has failed")
			log.Error("TCPDevDataServer has failed")
		}
	}()

	ln, err := net.Listen("tcp", server.LocalServer.Host+":"+fmt.Sprint(server.LocalServer.Port))

	for err != nil {

		for range server.Reconnect.C {
			ln, _ = net.Listen("tcp", server.LocalServer.Host+":"+fmt.Sprint(server.LocalServer.Port))
		}
		server.Reconnect.Stop()
	}

	for {
		conn, err := ln.Accept()
		if err == nil {
			go server.tcpDataHandler(conn)
		}
	}
}

func (server *TCPDevDataServer) tcpDataHandler(conn net.Conn) {
	var req entities.Request
	var res entities.Response
	for {
		err := json.NewDecoder(conn).Decode(&req)
		if err != nil {
			errors.Wrap(err, "Request decoding has failed")
			return
		}
		//sends resp struct from  devTypeHandler by channel;
		go server.devTypeHandler(req)

		res = entities.Response{
			Status: 200,
			Descr:  "Data has been delivered successfully",
		}
		err = json.NewEncoder(conn).Encode(&res)
		if err != nil {
			errors.Wrap(err, "Response encoding has failed")
		}
	}
}

func (server TCPDevDataServer) devTypeHandler(req entities.Request) string {
	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	switch req.Action {
	case "update":
		data := IdentifyDevice(req.Meta.Type)
		if data == nil || !entities.ValidateMAC(req.Meta.MAC) {
			return string("Device request: unknown device type")
		}
		log.Println("Data has been received")

		data.SetDevData(&req, dbClient)
		go db.PublishWS(req, "devWS", server.DbClient)

	default:
		log.Println("Device request: unknown action")
		return string("Device request: unknown action")

	}
	return string("Device request correct")
}
