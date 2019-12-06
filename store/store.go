// Package store provides means for data storage and retrieving.
package store

import (
	"fmt"
	"github.com/kostiamol/centerms/store/model"
	"time"

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

func (s *store) GetCfg(id string) (*model.Cfg, error) {
	dev, err := model.DefineDevice(m.Type)
	if err != nil {
		return nil, fmt.Errorf("func DefineDevice: %s", err)
	}
	return dev.GetCfg()
}

func (s *store) SetCfg(id string, c *model.Cfg) error {
	dev, err := model.DefineDevice()
	if err != nil {
		return fmt.Errorf("func DefineDevice: %s", err)
	}
	return dev.SetCfg(c)
}

func (s *store) GetDefaultCfg(m *model.Meta) (*model.Cfg, error) {
	dev, err := model.DefineDevice(m.Type)
	if err != nil {
		return nil, fmt.Errorf("func DefineDevice: %s", err)
	}
	return dev.GetDefaultCfg()
}

func (s *store) SetMeta(m *model.Meta) error {
	dev, err := model.DefineDevice(m.Type)
	if err != nil {
		return fmt.Errorf("func DefineDevice: %s", err)
	}
	return dev.SetMeta(m)
}

func (s *store) IsRegistered(m *model.Meta) (bool, error) {
	dev, err := model.DefineDevice(m.Type)
	if err != nil {
		return false, fmt.Errorf("func DefineDevice: %s", err)
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

func (s *store) SaveData(d *model.Data) error {
	dev, err := model.DefineDevice(d.Meta.Type)
	if err != nil {
		return fmt.Errorf("func DefineDevice: %s", err)
	}
	return dev.SaveData(d)
}
