package entities

import (
	"encoding/json"
	"time"

	consul "github.com/hashicorp/consul/api"
)

const (
	DevDataSubject            = "dev_data"
	DevConfigSubject          = "dev_config"
	timeForRoutineTermination = time.Second * 3
)

// Query the Consul for services:
// dig +noall +answer @127.0.0.1 -p 8600 myCoolServiceName.service.dc1.consul
// curl localhost:8500/v1/health/service/myCoolServiceName?passing

type Externer interface {
	Check() (bool, error)
	UpdateTTL(check func() (bool, error))
}

type Notifier interface {
	Publish(msg interface{}, channel string) (int64, error)
	Subscribe(cn chan []byte, subject ...string) error
}

type DevDataStorager interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id string) (*DevData, error)
	SaveDevData(req *SaveDevDataReq) error
	GetDevMeta(id string) (*DevMeta, error)
	SetDevMeta(meta *DevMeta) error
}

type DevConfigStorager interface {
	GetDevConfig(id string) (*DevConfig, error)
	SetDevConfig(id string, config *DevConfig) error
	GetDevDefaultConfig(meta *DevMeta) (*DevConfig, error)
	DevIsRegistered(meta *DevMeta) (bool, error)
}

type Storager interface {
	Externer
	Notifier
	DevDataStorager
	DevConfigStorager
	Init() error
	CreateConn() (Storager, error)
	CloseConn() error
}

// todo: type DeviceID uint
// todo: type InternalService struct + type Interner interface
// todo: change port type from string to uint

type Address struct {
	Host string
	Port string
}

type ExternalService struct {
	Addr        Address
	Name        string
	TTL         time.Duration
	ConsulAgent *consul.Agent
}

type Subscription struct {
	ChanName string
	Channel  chan []byte
}

type SaveDevDataReq struct {
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
	<-time.NewTimer(timeForRoutineTermination).C
}

func (c *ServiceController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
