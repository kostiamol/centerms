package entities

import (
	"encoding/json"
	"time"
)

const (
	DevDataChan   = "devDataChan"
	DevConfigChan = "devConfigChan"
)

type Notifier interface {
	Publish(channel string, msg interface{}) (int64, error)
	Subscribe(c chan []string, channel ...string) error
}

type DevDataDriver interface {
	GetDevsData() ([]DevData, error)
	GetDevData(m *DevMeta) (*DevData, error)
	SaveDevData(r *Request) error
}

type DevConfigDriver interface {
	GetDevConfig(m *DevMeta) (*DevConfig, error)
	SetDevConfig(m *DevMeta, c *DevConfig) error
	GetDevDefaultConfig(m *DevMeta) (*DevConfig, error)
	DevIsRegistered(m *DevMeta) (bool, error)
}

type DevStorage interface {
	Notifier
	DevDataDriver
	DevConfigDriver
	SetServer(s *Server) error
	CreateConn() (DevStorage, error)
	CloseConn() error
}

type Device interface {
}

type Server struct {
	Host string
	Port uint
}

type Request struct {
	Action string          `json:"action"`
	Time   int64           `json:"time"`
	Meta   DevMeta         `json:"meta"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	Status int    `json:"status"`
	Descr  string `json:"descr"`
}

type DevConfig struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

type DevData struct {
	Site string              `json:"site"`
	Meta DevMeta             `json:"meta"`
	Data map[string][]string `json:"data"`
}

type ServersController struct {
	StopChan chan struct{}
}

func (c *ServersController) Wait() {
	<-c.StopChan
	<-time.NewTimer(time.Second * 3).C
}

func (c *ServersController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
