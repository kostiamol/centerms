// Package entities provides definitions for common basic entities
// and some of their functions.
package entities

import (
	"encoding/json"
	"time"
)

// Inner channels for data exchange.
const (
	DevDataChan   = "dev_data"
	DevConfigChan = "dev_config"
)

// Inner events.
const (
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

// Externer is an external towards center entity which it depends on. The entity must provide method for checking it's
// state and updating TTL for integration with service discovery system such as Consul.
type Externer interface {
	Check() (bool, error)
	UpdateTTL(check func() (bool, error))
}

// Notifier is an entity that notifies subscribers with published messages.
type Notifier interface {
	Publish(msg interface{}, channel string) (int64, error)
	Subscribe(cn chan []byte, channel ...string)
}

// DevDataStorager deals with device data.
type DevDataStorager interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id DevID) (*DevData, error)
	SaveDevData(d *DevData) error
	GetDevMeta(id DevID) (*DevMeta, error)
	SetDevMeta(m *DevMeta) error
}

// DevConfigStorager deals with device configs.
type DevConfigStorager interface {
	GetDevConfig(id DevID) (*DevConfig, error)
	SetDevConfig(id DevID, c *DevConfig) error
	GetDevDefaultConfig(m *DevMeta) (*DevConfig, error)
	DevIsRegistered(m *DevMeta) (bool, error)
}

// Storager is an external entity for storing device data and configs and subscribing/publishing internal for the
// center data.
type Storager interface {
	Externer
	Notifier
	DevDataStorager
	DevConfigStorager
	Init() error
	CreateConn() (Storager, error)
	CloseConn() error
}

// DevID is used to device's id.
type DevID string

// Address is used to store IP address and an open port of the remote server.
type Address struct {
	Host string
	Port string
}

// Subscription is used to store channel name and chan for subscribing.
type Subscription struct {
	ChanName string
	Channel  chan []byte
}

// DevData is used to store time of the request, device's metadata and the data it transfers.
type DevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// DevConfig holds device's MAC address and config.
type DevConfig struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

// DevMeta is used to store device metadata: it's type, name (model) and MAC address.
type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

// ServiceController is used to store StopChan that allows to terminate all the services that listen the channel.
type ServiceController struct {
	StopChan chan struct{}
}

const timeForRoutineTermination = time.Second * 3

// Wait waits until StopChan will be closed and then makes a pause for the amount seconds defined in variable
// timeForRoutineTermination in order to give time for all the services to shutdown gracefully.
func (c *ServiceController) Wait() {
	<-c.StopChan
	<-time.NewTimer(timeForRoutineTermination).C
}

// Terminate closes StopChan to signal all the services to shutdown.
func (c *ServiceController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
