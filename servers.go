package main

import (
	"encoding/json"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
	"menteslibres.net/gosexy/redis"

	"strings"
)


func RunDBConnection() (*redis.Client, error) {
	client := redis.New()
	err := client.Connect(dbHost, dbPort)
	CheckError("RunDBConnection error: ", err)
	for err != nil {
		log.Errorln("1 Database: connection has failed:", err)
		time.Sleep(time.Second)
		err = client.Connect(dbHost, dbPort)
		if err == nil {
			continue
		}
		log.Errorln("err not nil")
	}

	return client, err
}

func runTCPServer() {
	var reconnect *time.Ticker

	ln, err := net.Listen(connType, connHost+":"+tcpConnPort)

	for err != nil {
		reconnect = time.NewTicker(time.Second * 1)
		for range reconnect.C {
			ln, _ = net.Listen(connType, connHost+":"+tcpConnPort)
		}
		reconnect.Stop()
	}

	for {
		conn, err := ln.Accept()
		if CheckError("TCP conn Accept", err) == nil {
			go tcpDataHandler(conn)
		}
	}
}

func reconnecting(dbClient *redis.Client) {
	var reconnect *time.Ticker
	for dbClient == nil {
		reconnect = time.NewTicker(time.Second * 1)
		for range reconnect.C {
			err := dbClient.Connect(dbHost, dbPort)
			log.Errorln("Database: connection has failed: %s\n", err)
		}
		return
	}
}

func runConfigServer(connType string, host string, port string) {

	var (
		dbClient  *redis.Client
		reconnect *time.Ticker
		pool      ConnectionPool
		messages  = make(chan []string)
	)

	pool.init()
	dbClient, err := RunDBConnection()
	CheckError("runConfigServer: RunDBConnection", err)

	go reconnecting(dbClient)
	defer dbClient.Close()

	ln, err := net.Listen(connType, host+":"+port)

	for err != nil {
		reconnect = time.NewTicker(time.Second * 1)
		for range reconnect.C {
			ln, _ = net.Listen(connType, connHost+":"+tcpConnPort)
		}
		reconnect.Stop()
	}
	go configSubscribe(dbClient, "configChan", messages, &pool)

	for {
		conn, err := ln.Accept()
		CheckError("TCP config conn Accept", err)
		go sendDefaultConfiguration(conn, &pool)
	}
}

func sendNewConfiguration(config DevConfig, pool *ConnectionPool) {

	connection := pool.getConn(config.MAC)
	if connection == nil {
		log.Error("Has not connection with mac:config.MAC  in connectionPool")
		return
	}

	// log.Println("mac in pool sendNewConfig", config.MAC)
	err := json.NewEncoder(connection).Encode(&config)

	if err != nil {
		pool.removeConn(config.MAC)
	}
	CheckError("sendNewConfig", err)
}

func sendDefaultConfiguration(conn net.Conn, pool *ConnectionPool) {
	// Send Default Configuration to Device
	var (
		req    Request
		config *DevConfig
	)

	dbClient, err := RunDBConnection()
	CheckError("DBConnection Error in ----> sendDefaultConfiguration", err)
	err = json.NewDecoder(conn).Decode(&req)
	CheckError("sendDefaultConfiguration JSON Decod", err)

	pool.addConn(conn, req.Meta.MAC)

	configInfo := req.Meta.MAC + ":" + "config" // key

	if ok, _ := dbClient.Exists(configInfo); ok {

		state, err := dbClient.HMGet(configInfo, "TurnedOn")
		CheckError("Get from DB error1: TurnedOn ", err)

		if  strings.Join(state, " ")!= "" {
			config = GetFridgeConfig(dbClient, configInfo,req.Meta.MAC)
			log.Println("Old Device with MAC: ",req.Meta.MAC, "detected.")
		}

	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config = CreateDefaultConfigToFridge()
		SetFridgeConfig(dbClient, configInfo, config)
	}

	err = json.NewEncoder(conn).Encode(&config)
	CheckError("sendDefaultConfiguration JSON enc", err)
	log.Warningln("Configuration has been successfully sent")
}

func CreateDefaultConfigToFridge() *DevConfig {
	return &DevConfig{
		TurnedOn:    true,
		StreamOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000,
	}
}
