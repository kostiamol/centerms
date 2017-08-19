package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	. "github.com/giperboloid/centerms/sys"

	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/devices"
	"reflect"
)

type TCPDevConfigServer struct {
	LocalServer         entities.Server
	Reconnect           *time.Ticker
	Pool                entities.ConnectionPool
	Messages            chan []string
	StopConfigSubscribe chan struct{}
	Controller          entities.RoutinesController
	DbClient            db.Client
}

func NewTCPDevConfigServer(local entities.Server, reconnect *time.Ticker, messages chan []string, stopConfigSubscribe chan struct{},
	controller entities.RoutinesController, dbClient db.Client) *TCPDevConfigServer {
	return &TCPDevConfigServer{
		LocalServer:         local,
		Reconnect:           reconnect,
		Messages:            messages,
		Controller:          controller,
		StopConfigSubscribe: stopConfigSubscribe,
		DbClient:            dbClient,
	}
}

func NewTCPDevConfigServerDefault(local entities.Server, controller entities.RoutinesController, dbClient db.Client) *TCPDevConfigServer {
	messages := make(chan []string)
	stopConfigSubscribe := make(chan struct{})
	reconnect := time.NewTicker(time.Second * 1)
	return NewTCPDevConfigServer(local, reconnect, messages, stopConfigSubscribe, controller, dbClient)
}

func (server *TCPDevConfigServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			server.StopConfigSubscribe <- struct{}{}
			server.Controller.Close()
			log.Error("TCPDevConfigServer Failed")
		}
	}()

	server.Pool.Init()

	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	ln, err := net.Listen("tcp", server.LocalServer.Host+":"+fmt.Sprint(server.LocalServer.Port))

	for err != nil {
		for range server.Reconnect.C {
			ln, _ = net.Listen("tcp", server.LocalServer.Host+":"+fmt.Sprint(server.LocalServer.Port))
		}
		server.Reconnect.Stop()
	}

	go server.configSubscribe("configChan", server.Messages, &server.Pool)

	for {
		conn, err := ln.Accept()
		CheckError("TCP config conn Accept", err)
		go server.sendDefaultConfiguration(conn, &server.Pool)
	}
}

func (server *TCPDevConfigServer) sendNewConfiguration(config entities.DevConfig, pool *entities.ConnectionPool) {
	connection := pool.GetConn(config.MAC)
	if connection == nil {
		log.Error("Has not connection with mac:config.MAC  in connectionPool")
		return
	}

	_, err := connection.Write(config.Data)

	if err != nil {
		pool.RemoveConn(config.MAC)
	}
	CheckError("sendNewConfig", err)
}

func (server *TCPDevConfigServer) sendDefaultConfiguration(conn net.Conn, pool *entities.ConnectionPool) {
	var (
		req    entities.Request
		config entities.DevConfig
		device devices.DevServerHandler
	)
	err := json.NewDecoder(conn).Decode(&req)
	CheckError("sendDefaultConfiguration JSON Decod", err)
	pool.AddConn(conn, req.Meta.MAC)
	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	device = IdentifyDevice(req.Meta.Type)
	log.Info(reflect.TypeOf(device))
	config.Data = device.SendDefaultConfigurationTCP(conn, dbClient, &req)

	_, err = conn.Write(config.Data)
	CheckError("sendDefaultConfiguration JSON enc", err)

	log.Warningln("Configuration has been successfully sent ", err)

}

func (server *TCPDevConfigServer) configSubscribe(roomID string, message chan []string, pool *entities.ConnectionPool) {
	dbClient := server.DbClient.NewDBConnection()
	defer dbClient.Close()

	dbClient.Subscribe(message, roomID)
	for {
		var config entities.DevConfig
		select {
		case msg := <-message:
			if msg[0] == "message" {
				err := json.Unmarshal([]byte(msg[2]), &config)
				CheckError("configSubscribe: unmarshal", err)
				go server.sendNewConfiguration(config, pool)
			}
		case <-server.StopConfigSubscribe:
			return
		}
	}
}
