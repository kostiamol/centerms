package svc

import (
	"time"
)

// Inner channels for data exchange.
const (
	DevDataChan = "dev_data"
	DevCfgChan  = "dev_cfg"
)

// devDataStorer deals with device data.
type devDataStorer interface {
	GetDevsData() ([]DevData, error)
	GetDevData(DevID) (*DevData, error)
	SaveDevData(*DevData) error
	GetDevMeta(DevID) (*DevMeta, error)
	SetDevMeta(*DevMeta) error
}

// devCfgStorer deals with device configurations.
type devCfgStorer interface {
	GetDevCfg(DevID) (*DevCfg, error)
	SetDevCfg(DevID, *DevCfg) error
	GetDevDefaultCfg(*DevMeta) (*DevCfg, error)
	DevIsRegistered(*DevMeta) (bool, error)
}

// Storer is an external svc for storing device data and configs and subscribing/publishing internal for the
// center data.
type Storer interface {
	devDataStorer
	devCfgStorer
	Init() error
	CreateConn() (Storer, error)
	CloseConn() error
	Publish(msg interface{}, channel string) (int64, error)
	Subscribe(cn chan []byte, channel ...string)
}

// Addr is used to store IP address and an open port of the remote server.
type Addr struct {
	Host string
	Port int
}

// DevID is used for device's id.
type DevID string

// subscription is used to store channel name and chan for subscribing.
type subscription struct {
	ChanName string
	Chan     chan []byte
}

// Ctrl is used to store StopChan that allows to terminate all the services that listen the channel.
type Ctrl struct {
	StopChan chan struct{}
}

const timeForRoutineTermination = time.Second * 3

// Wait waits until StopChan will be closed and then makes a pause for the amount seconds defined in variable
// timeForRoutineTermination in order to give time for all the services to shutdown gracefully.
func (c *Ctrl) Wait() {
	<-c.StopChan
	<-time.NewTimer(timeForRoutineTermination).C
}

// Terminate closes StopChan to signal all the services to shutdown.
func (c *Ctrl) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
