// Package store provides means for data storage and retrieving.
package store

import (
	"fmt"
	"time"

	"github.com/kostiamol/centerms/store/model"

	"github.com/kostiamol/centerms/cfg"
)

type (
	// store is used to provide a storage for
	store struct {
		addr cfg.Addr
	}

	// Cfg is used to initialize an instance of store.
	Cfg struct {
		Addr             cfg.Addr
		Password         string
		MaxIdlePoolConns uint32
		IdleTimeout      time.Duration
	}
)

// New creates a new instance of store.
func New(c *Cfg) (*store, error) { // nolint
	return nil, nil
}

func (s *store) InitCfg(m *model.Meta) (*model.Cfg, error) {
	return nil, nil
}

func (s *store) GetCfg(id string, t model.Type) (*model.Cfg, error) {
	dev, err := model.NewDevice(id, t)
	if err != nil {
		return nil, fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.GetCfg()
}

func (s *store) SetCfg(id string, t model.Type, c *model.Cfg) error {
	dev, err := model.NewDevice(id, t)
	if err != nil {
		return fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.SetCfg(c)
}

func (s *store) GetDefaultCfg(id string, t model.Type) (*model.Cfg, error) {
	dev, err := model.NewDevice(id, t)
	if err != nil {
		return nil, fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.GetDefaultCfg()
}

func (s *store) SetMeta(id string, m *model.Meta) error {
	dev, err := model.NewDevice(id, m.Type)
	if err != nil {
		return fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.SetMeta(m)
}

func (s *store) IsRegistered(id string, t model.Type) (bool, error) {
	dev, err := model.NewDevice(id, t)
	if err != nil {
		return false, fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.IsRegistered()
}

func (s *store) GetDevsData() ([]model.Data, error) {
	// todo
	return nil, nil
}

func (s *store) GetDevData(id string) (*model.Data, error) {
	// todo
	return nil, nil
}

func (s *store) SaveData(id string, t model.Type, d *model.Data) error {
	dev, err := model.NewDevice(id, t)
	if err != nil {
		return fmt.Errorf("func NewDevice: %s", err)
	}
	return dev.SaveData(d)
}
