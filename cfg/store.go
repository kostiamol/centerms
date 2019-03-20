package cfg

import (
	"fmt"
)

// Store holds store configuration.
type Store struct {
	Host     string
	Port     uint64
	Password string
}

func (s Store) validate() error {
	if s.Host == "" {
		return fmt.Errorf("StoreHost is missing")
	}
	if s.Port == 0 {
		return fmt.Errorf("StorePort is missing")
	}
	if s.Password == "" {
		return fmt.Errorf("LogLevel is missing")
	}
	return nil
}
