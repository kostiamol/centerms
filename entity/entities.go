// Package entity provides definitions for common basic entities
// and some of their functions.
package entity

import (
	"encoding/json"
)

// Inner channels for data exchange.
const (
	DevDataChan = "dev_data"
	DevCfgChan  = "dev_cfg"
)

// Inner events.
const (
	EventCfgPatchCreated = "conf_patch_created"
	EventDevRegistered   = "dev_registered"
	EventMSTerminated    = "ms_terminated"
	EventPanic           = "panic"
	EventStoreInit       = "store_init"
	EventSVCStarted      = "svc_started"
	EventSVCShutdown     = "svc_shutdown"
	EventWSConnAdded     = "ws_conn_added"
	EventWSConnRemoved   = "ws_conn_removed"
	EventUpdConsulStatus = "upd_consul_status"
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

// DevDataStorer deals with device data.
type DevDataStorer interface {
	GetDevsData() ([]DevData, error)
	GetDevData(id DevID) (*DevData, error)
	SaveDevData(d *DevData) error
	GetDevMeta(id DevID) (*DevMeta, error)
	SetDevMeta(m *DevMeta) error
}

// DevCfgStorer deals with device configurations.
type DevCfgStorer interface {
	GetDevCfg(id DevID) (*DevCfg, error)
	SetDevCfg(id DevID, c *DevCfg) error
	GetDevDefaultCfg(m *DevMeta) (*DevCfg, error)
	DevIsRegistered(m *DevMeta) (bool, error)
}

// Storer is an external entity for storing device data and configs and subscribing/publishing internal for the
// center data.
type Storer interface {
	Externer
	Notifier
	DevDataStorer
	DevCfgStorer
	Init() error
	CreateConn() (Storer, error)
	CloseConn() error
}

// DevID is used to device's id.
type DevID string

// Addr is used to store IP address and an open port of the remote server.
type Addr struct {
	Host string
	Port int
}

// Subscription is used to store channel name and chan for subscribing.
type Subscription struct {
	ChanName string
	Chan     chan []byte
}

// DevData is used to store time of the request, device's metadata and the data it transfers.
type DevData struct {
	Time int64           `json:"time"`
	Meta DevMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// DevCfg holds device's MAC address and config.
type DevCfg struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

// DevMeta is used to store device metadata: it's type, name (model) and MAC address.
type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

// User is used to authenticate requests
type User struct {
	UUID     string `json:"uuid" form:"-"`
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}
