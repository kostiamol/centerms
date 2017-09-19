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
	s.Log.Infof("DevDataServer has started on host: %s, port: %d", s.Server.Host, s.Server.Port)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DevDataServer: Run(): panic: %s", r)
			cancel()
			s.gracefulHalt()
		}
	}()

	go s.handleTermination()

	ln, err := net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
	if err != nil {
		s.Log.Errorf("DevDataServer: Run(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range s.Reconnect.C {
			ln, err = net.Listen("tcp", s.Server.Host+":"+fmt.Sprint(s.Server.Port))
			if err != nil {
				s.Log.Errorf("DevDataServer: Run(): Listen() has failed: %s", err)
			}
		}
		s.Reconnect.Stop()
	}

	for {
		if conn, err := ln.Accept(); err == nil {
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
	var req entities.Request
	for {
		if err := json.NewDecoder(c).Decode(&req); err != nil {
			s.Log.Errorf("DevDataServer: devDataHandler(): Request decoding has failed: %s", err)
			return
		}

		go s.saveDevData(ctx, &req)

		resp := entities.Response{
			Status: 200,
			Descr:  "DevData has been delivered",
		}

		if err := json.NewEncoder(c).Encode(&resp); err != nil {
			s.Log.Errorf("DevDataServer: devDataHandler(): Response encoding has failed: %s", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevDataServer) saveDevData(ctx context.Context, r *entities.Request) {
	s.Log.Infof("Saving data for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		s.Log.Errorf("DevDataServer: saveDevData(): storage connection hasn't been established: %s", err)
		return
	}
	defer cn.CloseConn()

	if err = cn.SetDevData(r); err != nil {
		s.Log.Errorf("DevDataServer: SetDevData() has failed: %s", err)
		return
	}

	go s.publishWS(ctx, r, "devWS")

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (s *DevDataServer) publishWS(ctx context.Context, r *entities.Request, roomID string) error {
	b, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "DevDataServer: publishWS(): Request marshalling has failed")
		return err
	}

	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		errors.Wrap(err, "DevDataServer: publishWS(): storage connection hasn't been established")
		return err
	}
	defer cn.CloseConn()

	if _, err = cn.Publish(roomID, b); err != nil {
		errors.Wrap(err, "DevDataServer: publishWS(): publishing has failed")
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}
