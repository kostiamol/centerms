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
		return fmt.Errorf("store host env var is missing")
	}
	if s.Addr.Port == 0 {
		return fmt.Errorf("store port env var is missing")
	}
	if s.Password == "" {
		return fmt.Errorf("store password env var is missing")
	}
	if s.IdleTimeout == 0 {
		return fmt.Errorf("store idle timeout env var is missing")
	}
	if s.MaxIdlePoolConns == 0 {
		return fmt.Errorf("store max idle pool conns env var is missing")
	}
	return nil
}
