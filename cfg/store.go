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
		return fmt.Errorf("storeHost is missing")
	}
	if s.Port == 0 {
		return fmt.Errorf("storePort is missing")
	}
	if s.Password == "" {
		return fmt.Errorf("logLevel is missing")
	}
	return nil
}
