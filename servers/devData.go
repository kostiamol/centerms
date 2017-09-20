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
		if cn, err := ln.Accept(); err == nil {
			s.Log.Info("DevDataServer: connection with device has been established")
			go s.devDataHandler(ctx, cn)
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
		select {
		case <-ctx.Done():
			return
		default:
			if err := json.NewDecoder(c).Decode(&req); err != nil {
				s.Log.Errorf("DevDataServer: devDataHandler(): Request decoding has failed: %s", err)
				return
			}

			go s.saveDevData(&req)

			resp := entities.Response{
				Status: 200,
				Descr:  "DevData has been delivered",
			}

			if err := json.NewEncoder(c).Encode(&resp); err != nil {
				s.Log.Errorf("DevDataServer: devDataHandler(): Response encoding has failed: %s", err)
				return
			}
		}
	}
}

func (s *DevDataServer) saveDevData(r *entities.Request) {
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

	//s.Log.Infof("save data for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	go s.publishDevData(r, entities.DevDataChan)
}

func (s *DevDataServer) publishDevData(r *entities.Request, channel string) error {
	b, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "DevDataServer: publishDevData(): Request marshalling has failed")
	}

	cn, err := s.DevStorage.CreateConn()
	if err != nil {
		return errors.Wrap(err, "DevDataServer: publishDevData(): storage connection hasn't been established")
	}
	defer cn.CloseConn()

	if _, err = cn.Publish(channel, b); err != nil {
		return errors.Wrap(err, "DevDataServer: publishDevData(): publishing has failed")
	}

	//s.Log.Infof("publish DevData for device with TYPE: [%s]; NAME: [%s]; MAC: [%s]", r.Meta.Type, r.Meta.Name, r.Meta.MAC)
	return nil
}
