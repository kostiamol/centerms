package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type DevConfigServer struct {
	Server              entities.Server
	DevStore            entities.DevStore
	Controller          entities.RoutinesController
	Log                 *logrus.Logger
	Reconnect           *time.Ticker
	Pool                entities.ConnPool
	Messages            chan []string
	StopConfigSubscribe chan struct{}
}

func NewDevConfigServer(s entities.Server, ds entities.DevStore, c entities.RoutinesController, l *logrus.Logger,
	r *time.Ticker, msgs chan []string, stopConfigSubscribe chan struct{}) *DevConfigServer {
	l.Out = os.Stdout
	return &DevConfigServer{
		Server:              s,
		DevStore:            ds,
		Controller:          c,
		Log:                 l,
		Reconnect:           r,
		Messages:            msgs,
		StopConfigSubscribe: stopConfigSubscribe,
	}
}

func (s *DevConfigServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			s.StopConfigSubscribe <- struct{}{}
			errors.New("DevConfigServer: Run(): panic leads to halt")
			s.gracefulHalt()
			s.Controller.Close()
		}
	}()

	s.Pool.Init()
	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		errors.Wrap(err, "DevConfigServer: Run(): Listen() has failed")
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				errors.Wrap(err, "DevConfigServer: Run(): Listen() has failed")
			}
		}
		s.Reconnect.Stop()
	}

	go s.configSubscribe("configChan", s.Messages, &s.Pool)

	for {
		conn, err := ln.Accept()
		if err != nil {
			errors.Wrap(err, "DevConfigServer: Run(): Accept() has failed")
		}
		go s.sendDefaultConfig(conn, &s.Pool)
	}
}

func (s *DevConfigServer) gracefulHalt() {
	s.DevStore.CloseConn()
}

func (s *DevConfigServer) sendNewConfig(c *entities.DevConfig, p *entities.ConnPool) {
	connection := p.GetConn(c.MAC)
	if connection == nil {
		errors.New("DevConfigServer: sendNewConfig(): there isn't such a connection in pool")
		return
	}

	_, err := connection.Write(c.Data)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendNewConfig(): DevConfig.Data writing has failed")
		p.RemoveConn(c.MAC)
	}
}

func (s *DevConfigServer) sendDefaultConfig(c net.Conn, pool *entities.ConnPool) {
	var req entities.Request
	err := json.NewDecoder(c).Decode(&req)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): Request marshalling has failed")
	}
	pool.AddConn(c, req.Meta.MAC)

	conn, err := s.DevStore.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConn()

	config, err := conn.GetDevDefaultConfig(&req.Meta)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): DevConfig extraction has failed")
	}
	if config != nil {
		conn.SetDevConfig(&req.Meta, config)
	}

	_, err = c.Write(config.Data)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): DevConfig.Data writing has failed")
	}
}

func (s *DevConfigServer) configSubscribe(roomID string, msg chan []string, pool *entities.ConnPool) {
	conn, err := s.DevStore.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: configSubscribe(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConn()

	conn.Subscribe(msg, roomID)
	for {
		var c entities.DevConfig
		select {
		case msg := <-msg:
			if msg[0] == "message" {
				err := json.Unmarshal([]byte(msg[2]), &c)
				if err != nil {
					errors.Wrap(err, "DevConfigServer: configSubscribe(): DevConfig unmarshalling has failed")
				}
				go s.sendNewConfig(&c, pool)
			}
		case <-s.StopConfigSubscribe:
			return
		}
	}
}
