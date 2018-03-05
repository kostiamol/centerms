package entities

import (
	"encoding/json"
	"time"
)

const (
	DevDataChan               = "dev_data"
	DevConfigChan             = "dev_config"
	timeForRoutineTermination = time.Second * 3

	EventConfigPatchCreated = "config_patch_created"
	EventDevRegistered      = "dev_registered"
	EventMSTerminated       = "ms_terminated"
	EventPanic              = "panic"
	EventStorageInit        = "storage_init"
	EventSVCStarted         = "svc_started"
	EventSVCShutdown        = "svc_shutdown"
	EventWSConnAdded        = "ws_conn_added"
	EventWSConnRemoved      = "ws_conn_removed"
	EventUpdConsulStatus    = "upd_consul_status"
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
	Subscribe(cn chan []byte, channel ...string)
}

type DevDataStorager interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id DevID) (*DevData, error)
	SaveDevData(d *RawDevData) error
	GetDevMeta(id DevID) (*DevMeta, error)
	SetDevMeta(m *DevMeta) error
}

type DevConfigStorager interface {
	GetDevConfig(id DevID) (*DevConfig, error)
	SetDevConfig(id DevID, config *DevConfig) error
	GetDevDefaultConfig(m *DevMeta) (*DevConfig, error)
	DevIsRegistered(m *DevMeta) (bool, error)
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

type DevID string

type Address struct {
	Host string
	Port string
}

type Subscription struct {
	ChanName string
	Channel  chan []byte
}

type RawDevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

type DevData struct {
	Meta DevMeta             `json:"meta"`
	Data map[string][]string `json:"data"`
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
