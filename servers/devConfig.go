package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type DevConfigServer struct {
	LocalServer         entities.Server
	Reconnect           *time.Ticker
	Pool                entities.ConnPool
	Messages            chan []string
	StopConfigSubscribe chan struct{}
	Controller          entities.RoutinesController
	Storage             entities.Storage
}

func NewDevConfigServer(s entities.Server, r *time.Ticker, c entities.RoutinesController, st entities.Storage,
	msgs chan []string, stopConfigSubscribe chan struct{}) *DevConfigServer {
	return &DevConfigServer{
		LocalServer:         s,
		Reconnect:           r,
		Messages:            msgs,
		Controller:          c,
		StopConfigSubscribe: stopConfigSubscribe,
		Storage:             st,
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

func (s *DevConfigServer) sendNewConfiguration(config entities.DevConfig, pool *entities.ConnPool) {
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

func (s *DevConfigServer) sendDefaultConfiguration(c net.Conn, pool *entities.ConnPool) {
	var (
		req    entities.Request
		config entities.DevConfig
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

	config.Data, err = s.Storage.SendDevDefaultConfig(c, &req)
	if err != nil {
		errors.Wrap(err, "config sending has failed")
	}

	_, err = c.Write(config.Data)
	if err != nil {
		errors.Wrap(err, "data writing has failed")
	}

	log.Warningln("Configuration has been successfully sent ", err)
}

func (s *DevConfigServer) configSubscribe(roomID string, message chan []string, pool *entities.ConnPool) {
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
