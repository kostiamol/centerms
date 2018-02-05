package entities

import (
	"encoding/json"
	"time"
)

const (
	DevDataSubject   = "dev_data"
	DevConfigSubject = "dev_config"
)

// type DeviceID uint

type Notifier interface {
	Publish(subject string, message interface{}) (int64, error)
	Subscribe(cn chan []string, subject ...string) error
}

type DevDataDriver interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id string) (*DevData, error)
	SaveDevData(req *SaveDevDataRequest) error
	GetDevMeta(id string) (*DevMeta, error)
	SetDevMeta(meta *DevMeta) error
}

type DevConfigDriver interface {
	GetDevConfig(id string) (*DevConfig, error)
	SetDevConfig(id string, config *DevConfig) error
	GetDevDefaultConfig(meta *DevMeta) (*DevConfig, error)
	DevIsRegistered(meta *DevMeta) (bool, error)
}

type Storage interface {
	Notifier
	DevDataDriver
	DevConfigDriver
	SetServer(srv Server) error
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
