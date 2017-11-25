package entities

import (
	"encoding/json"
	"time"
)

const (
	DevDataSubject   = "devData"
	DevConfigSubject = "devConfig"
)

type DeviceID uint

type Notifier interface {
	Publish(subject string, message interface{}) (int64, error)
	Subscribe(cn chan []string, subject ...string) error
}

type DevDataDriver interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id string) (*DevData, error)
	SaveDevData(r *SaveDevDataRequest) error
	GetDevMeta(id string) (*DevMeta, error)
	SetDevMeta(m *DevMeta) error
}

type DevConfigDriver interface {
	GetDevConfig(id string) (*DevConfig, error)
	SetDevConfig(id string, c *DevConfig) error
	GetDevDefaultConfig(m *DevMeta) (*DevConfig, error)
	DevIsRegistered(m *DevMeta) (bool, error)
}

type Storage interface {
	Notifier
	DevDataDriver
	DevConfigDriver
	SetServer(s Server) error
	CreateConn() (Storage, error)
	CloseConn() error
}

type Server struct {
	Host string
	Port string
}

type Subscription struct {
	Subject string
	Channel chan []string
}

type SaveDevDataRequest struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
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

type ServiceController struct {
	StopChan chan struct{}
}

func (c *ServiceController) Wait() {
	<-c.StopChan
	<-time.NewTimer(time.Second * 3).C
}

func (c *ServiceController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
