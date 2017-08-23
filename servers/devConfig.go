package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	"reflect"
	"github.com/pkg/errors"
	"github.com/giperboloid/centerms/storages/redis"
)

type DevConfigServer struct {
	LocalServer         entities.Server
	Reconnect           *time.Ticker
	Pool                entities.ConnectionPool
	Messages            chan []string
	StopConfigSubscribe chan struct{}
	Controller          entities.RoutinesController
	Storage             entities.Storage
}

func NewDevConfigServer(serv entities.Server, reconnect *time.Ticker, messages chan []string, stopConfigSubscribe chan struct{},
	c entities.RoutinesController, s entities.Storage) *DevConfigServer {
	return &DevConfigServer{
		LocalServer:         serv,
		Reconnect:           reconnect,
		Messages:            messages,
		Controller:          c,
		StopConfigSubscribe: stopConfigSubscribe,
		Storage:             s,
	}
}

func NewDevConfigServerDefault(serv entities.Server, c entities.RoutinesController, s entities.Storage) *DevConfigServer {
	messages := make(chan []string)
	stopConfigSubscribe := make(chan struct{})
	reconnect := time.NewTicker(time.Second * 1)
	return &DevConfigServer{
		LocalServer:         serv,
		Reconnect:           reconnect,
		Messages:            messages,
		Controller:          c,
		StopConfigSubscribe: stopConfigSubscribe,
		Storage:             s,
	}
}

func (s *DevConfigServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			s.StopConfigSubscribe <- struct{}{}
			s.Controller.Close()
			log.Error("DevConfigServer Failed")
		}
	}()

	s.Pool.Init()

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	ln, err := net.Listen("tcp", s.LocalServer.Host+":"+fmt.Sprint(s.LocalServer.Port))

	for err != nil {
		for range s.Reconnect.C {
			ln, _ = net.Listen("tcp", s.LocalServer.Host+":"+fmt.Sprint(s.LocalServer.Port))
		}
		s.Reconnect.Stop()
	}

	go s.configSubscribe("configChan", s.Messages, &s.Pool)

	for {
		conn, err := ln.Accept()
		if err != nil {
			errors.Wrap(err, "tcp connection acceptance has failed")
		}
		go s.sendDefaultConfiguration(conn, &s.Pool)
	}
}

func (s *DevConfigServer) sendNewConfiguration(config entities.DevConfig, pool *entities.ConnectionPool) {
	connection := pool.GetConn(config.MAC)
	if connection == nil {
		log.Error("Has not connection with mac:config.MAC  in connectionPool")
		return
	}

	_, err := connection.Write(config.Data)
	if err != nil {
		errors.Wrap(err, "Data writing has failed")
		pool.RemoveConn(config.MAC)
	}
}

func (s *DevConfigServer) sendDefaultConfiguration(c net.Conn, pool *entities.ConnectionPool) {
	var (
		req    entities.Request
		config entities.DevConfig
		device devices.DevServerHandler
	)
	err := json.NewDecoder(c).Decode(&req)
	if err != nil {
		errors.Wrap(err, "Request marshalling has failed")
	}
	pool.AddConn(c, req.Meta.MAC)

	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	device = IdentifyDevice(req.Meta.Type)
	log.Info(reflect.TypeOf(device))
	config.Data = s.Storage.SendDefaultConfiguration(c, conn, &req)

	_, err = c.Write(config.Data)
	if err != nil {
		errors.Wrap(err, "data writing has failed")
	}

	log.Warningln("Configuration has been successfully sent ", err)

}

func (s *DevConfigServer) configSubscribe(roomID string, message chan []string, pool *entities.ConnectionPool) {
	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
		return
	}
	defer conn.CloseConnection()

	conn.Subscribe(message, roomID)
	for {
		var config entities.DevConfig
		select {
		case msg := <-message:
			if msg[0] == "message" {
				err := json.Unmarshal([]byte(msg[2]), &config)
				if err != nil {
					errors.Wrap(err, "DevConfig marshalling has failed")
				}
				go s.sendNewConfiguration(config, pool)
			}
		case <-s.StopConfigSubscribe:
			return
		}
	}
}
