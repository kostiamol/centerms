package servers

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/entities"
	storages "github.com/giperboloid/centerms/storages/redis"
	"github.com/pkg/errors"
)

type DevDataServer struct {
	LocalServer entities.Server
	Reconnect   *time.Ticker
	Controller  entities.RoutinesController
	Storage     entities.Storage
}

func NewDevDataServer(serv entities.Server, reconnect *time.Ticker, c entities.RoutinesController, s entities.Storage) *DevDataServer {
	return &DevDataServer{
		LocalServer: serv,
		Reconnect:   reconnect,
		Controller:  c,
		Storage:     s,
	}
}

func NewDevDataServerDefault(serv entities.Server, c entities.RoutinesController, s entities.Storage) *DevDataServer {
	reconnect := time.NewTicker(time.Second * 1)
	return NewDevDataServer(serv, reconnect, c, s)
}

func (s *DevDataServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			s.Controller.Close()
			errors.New("DevDataServer has failed")
			log.Error("DevDataServer has failed")
		}
	}()

	ln, err := net.Listen("tcp", s.LocalServer.Host+":"+fmt.Sprint(s.LocalServer.Port))

	for err != nil {

		for range s.Reconnect.C {
			ln, _ = net.Listen("tcp", s.LocalServer.Host+":"+fmt.Sprint(s.LocalServer.Port))
		}
		s.Reconnect.Stop()
	}

	for {
		conn, err := ln.Accept()
		if err == nil {
			go s.tcpDataHandler(conn)
		}
	}
}

func (s *DevDataServer) tcpDataHandler(conn net.Conn) {
	var req entities.Request
	var res entities.Response
	for {
		err := json.NewDecoder(conn).Decode(&req)
		if err != nil {
			errors.Wrap(err, "Request decoding has failed")
			return
		}
		//sends resp struct from  devTypeHandler by channel;
		go s.devTypeHandler(&req)

		res = entities.Response{
			Status: 200,
			Descr:  "Data has been delivered successfully",
		}
		err = json.NewEncoder(conn).Encode(&res)
		if err != nil {
			errors.Wrap(err, "Response encoding has failed")
		}
	}
}

func (s *DevDataServer) devTypeHandler(req *entities.Request) string {
	conn, err := s.Storage.CreateConnection()
	if err != nil {
		log.Errorln("db connection hasn't been established")
	}
	defer conn.CloseConnection()

	switch req.Action {
	case "update":
		data := IdentifyDevice(req.Meta.Type)
		if data == nil || !entities.ValidateMAC(req.Meta.MAC) {
			return string("Device request: unknown device type")
		}
		log.Println("Data has been received")

		s.Storage.SetDevData(&req, )
		go storages.PublishWS(req, "devWS", s.Storage)

	default:
		log.Println("Device request: unknown action")
		return string("Device request: unknown action")

	}
	return string("Device request correct")
}
