package cfg

import (
	"fmt"
	"time"
)

// Store holds store configuration.
type Store struct {
	Addr             Addr
	Password         string
	IdleTimeout      time.Duration
	MaxIdlePoolConns uint32
}

func (s Store) validate() error {
	if s.Addr.Host == "" {
		return fmt.Errorf("storeHost is missing")
	}
	if s.Addr.Port == 0 {
		return fmt.Errorf("storePort is missing")
	}
	if s.Password == "" {
		return fmt.Errorf("storePassword is missing")
	}
	if s.IdleTimeout == 0 {
		return fmt.Errorf("storeIdleTimeout is missing")
	}
	if s.MaxIdlePoolConns == 0 {
		return fmt.Errorf("storeMaxIdlePoolConns is missing")
	}
	return nil
}
