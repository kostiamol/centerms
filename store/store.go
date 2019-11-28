// Package store provides means for data storage and retrieving.
package store

import (
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
