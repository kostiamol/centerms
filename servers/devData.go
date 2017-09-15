package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"context"
)

type DevDataServer struct {
	Server     entities.Server
	DevStorage entities.DevStorage
	Controller entities.ServersController
	Log        *logrus.Logger
	Reconnect  *time.Ticker
}

func NewDevDataServer(s entities.Server, ds entities.DevStorage, c entities.ServersController,
	l *logrus.Logger, r *time.Ticker) *DevDataServer {
	return &DevDataServer{
		Server:     s,
		DevStorage: ds,
		Controller: c,
		Log:        l,
		Reconnect:  r,
	}
}

func (s *DevDataServer) Run() {
	s.Log.Infoln("DevDataServer has started")
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Error("DevDataServer: Run(): panic: ", r)
			cancel()
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

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

	for {
		conn, err := ln.Accept()
		if err == nil {
			go s.devDataHandler(ctx, conn)
		}
	}
}

func (s *DevDataServer) handleTermination() {
	for {
		select {
		case <-s.Controller.StopChan:
			s.gracefulHalt()
			return
		}
	}
}

func (s *DevDataServer) gracefulHalt() {
	s.DevStorage.CloseConn()
	s.Log.Infoln("DevDataServer has shut down")
	s.Controller.Terminate()
}

func (s *DevDataServer) devDataHandler(ctx context.Context, c net.Conn) {
	var r entities.Request
	for {
		err := json.NewDecoder(c).Decode(&r)
		if err != nil {
			errors.Wrap(err, "DevConfigServer: devDataHandler(): Request decoding has failed")
			return
		}

		go s.devTypeHandler(ctx, &r)

		resp := entities.Response{
			Status: 200,
			Descr:  "OK",
		}
		err = json.NewEncoder(c).Encode(&resp)
		if err != nil {
			errors.Wrap(err, "DevConfigServer: devDataHandler(): Response encoding has failed")
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevDataServer) devTypeHandler(ctx context.Context, r *entities.Request) {
	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: devTypeHandler(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	switch r.Action {
	case "update":
		conn.SetDevData(r)
		go s.publishWS(r, "devWS")
	default:
		errors.Wrap(err, "DevConfigServer: devTypeHandler(): unknown action")
	}

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevDataServer) publishWS(r *entities.Request, roomID string) error {
	pr, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "DevConfigServer: publishWS(): Request marshalling has failed")
	}

	conn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevConfigServer: publishWS(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	_, err = s.DevStorage.Publish(roomID, pr)
	if err != nil {
		errors.Wrap(err, "DevConfigServer: publishWS(): publishing has failed")
	}

	return err
}
