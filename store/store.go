// Package store provides means for data storage and retrieving.
package store

import (
	"time"

	"github.com/kostiamol/centerms/store/dev"

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

func (s *store) InitCfg(m *dev.Meta) (*dev.Cfg, error) {
	return nil, nil
}

func (s *store) GetCfg(id string) (*dev.Cfg, error) {
	return nil, nil
}

func (s *store) SetCfg(id string, c *dev.Cfg) error {
	return nil
}

func (s *store) GetDefaultCfg(*dev.Meta) (*dev.Cfg, error) {
	return nil, nil
}

func (s *store) SetMeta(*dev.Meta) error {
	return nil
}

func (s *store) IsRegistered(*dev.Meta) (bool, error) {
	return false, nil
}

func (s *store) GetDevsData() ([]dev.Data, error) {
	return nil, nil
}

func (s *store) GetDevData(id string) (*dev.Data, error) {
	return nil, nil
}

func (s *store) SaveData(*dev.Data) error {
	return nil
}
