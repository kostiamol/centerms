package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"os"

	"context"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type ConnPool struct {
	sync.Mutex
	conn map[string]net.Conn
}

func (p *ConnPool) Init() {
	p.Lock()
	defer p.Unlock()
	p.conn = make(map[string]net.Conn)
}

func (p *ConnPool) AddConn(conn net.Conn, key string) {
	p.Lock()
	p.conn[key] = conn
	defer p.Unlock()
}

func (p *ConnPool) GetConn(key string) net.Conn {
	p.Lock()
	defer p.Unlock()
	return p.conn[key]
}

func (p *ConnPool) RemoveConn(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.conn, key)
}

type DevConfigServer struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServersController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
	ConnPool   ConnPool
	Messages   chan []string
}

func NewDevConfigServer(s entities.Server, ds entities.DevStorage, c entities.ServersController, l *logrus.Logger,
	r *time.Ticker, msgs chan []string) *DevConfigServer {
	l.Out = os.Stdout

	return &DevConfigServer{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
		Messages:   msgs,
	}
}

func (s *DevConfigServer) Run() {
	s.Log.Infoln("DevConfigServer has started")
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Error("DevConfigServer: Run(): panic: ", r)
			cancel()
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

	s.ConnPool.Init()
	go s.configSubscribe(ctx, "configChan", s.Messages, &s.ConnPool)

	go s.listenForConnections(ctx)
}

func (s *DevConfigServer) handleTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.gracefulHalt()
			return
		}
	}
}

func (s *DevConfigServer) gracefulHalt() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DevConfigServer has shut down")
	s.Controller.Terminate()
}

func (s *DevConfigServer) listenForConnections(ctx context.Context) {
	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				errors.Wrap(err, "DevConfigServer: Run(): Listen() has failed")
			}
		}
		s.Reconnect.Stop()
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			errors.Wrap(err, "DevConfigServer: Run(): Accept() has failed")
		}
		go s.sendDefaultConfig(ctx, conn, &s.ConnPool)
	}

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevConfigServer) sendDefaultConfig(ctx context.Context, c net.Conn, cp *ConnPool) {
	var r entities.Request
	err := json.NewDecoder(c).Decode(&r)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): Request marshalling has failed")
	}
	cp.AddConn(c, r.Meta.MAC)

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendDefaultConfig(): storage connection hasn't been established")
		return
	}
	defer conn.CloseConn()

	var config *entities.DevConfig
	if ok, err := conn.DevIsRegistered(&r.Meta); ok {
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

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevConfigServer) configSubscribe(ctx context.Context, roomID string, msg chan []string, cp *ConnPool) {
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
				go s.sendNewConfig(ctx, &c, cp)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevConfigServer) sendNewConfig(ctx context.Context, c *entities.DevConfig, cp *ConnPool) {
	conn := cp.GetConn(c.MAC)
	if conn == nil {
		errors.New("DevConfigServer: sendNewConfig(): there isn't such a connection in pool")
		return
	}

	_, err := conn.Write(c.Data)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: sendNewConfig(): DevConfig.Data writing has failed")
		cp.RemoveConn(c.MAC)
	}

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
