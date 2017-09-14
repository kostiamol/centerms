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
	DevStorage          entities.DevStorage
	Controller          entities.RoutinesController
	Log                 *logrus.Logger
	Reconnect           *time.Ticker
	Pool                entities.ConnPool
	Messages            chan []string
	StopConfigSubscribe chan struct{}
}

func NewDevConfigServer(s entities.Server, ds entities.DevStorage, c entities.RoutinesController, l *logrus.Logger,
	r *time.Ticker, msgs chan []string, stopConfigSubscribe chan struct{}) *DevConfigServer {
	l.Out = os.Stdout
	return &DevConfigServer{
		Server:              s,
		DevStorage:          ds,
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
	s.DevStorage.CloseConn()
}

func (s *DevConfigServer) sendNewConfig(c *entities.DevConfig, p *entities.ConnPool) {
	conn := p.GetConn(c.MAC)
	if conn == nil {
		errors.New("DevConfigServer: sendNewConfig(): there isn't such a connection in pool")
		return
	}

	_, err := conn.Write(c.Data)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendNewConfig(): DevConfig.Data writing has failed")
		p.RemoveConn(c.MAC)
	}
}

func (s *DevConfigServer) sendDefaultConfig(c net.Conn, p *entities.ConnPool) {
	var r entities.Request
	err := json.NewDecoder(c).Decode(&r)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): Request marshalling has failed")
	}
	p.AddConn(c, r.Meta.MAC)

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConn()

	var config *entities.DevConfig
	if ok, err := conn.DevIsRegistered(&r.Meta); !ok {
		if err != nil {
			errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): DevIsRegistered() has failed")
		}

		config, err = conn.GetDevConfig(&r.Meta)
		if err != nil {
			errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): GetDevConfig() has failed")
		}
	} else {
		config, err = conn.GetDevDefaultConfig(&r.Meta)
		if err != nil {
			errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): GetDevDefaultConfig() has failed")
		}
		err = conn.SetDevConfig(&r.Meta, config)
		if err != nil {
			errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): SetDevConfig() has failed")
		}
	}

	_, err = c.Write(config.Data)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): DevConfig.Data writing has failed")
	}
}

func (s *DevConfigServer) configSubscribe(roomID string, msg chan []string, p *entities.ConnPool) {
	conn, err := s.DevStorage.CreateConn()
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
				go s.sendNewConfig(&c, p)
			}
		case <-s.StopConfigSubscribe:
			return
		}
	}
}
