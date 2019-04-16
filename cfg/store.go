package cfg

import (
	"fmt"
)

// Store holds store configuration.
type Store struct {
	Addr     Addr
	Password string
}

func (s Store) validate() error {
	if s.Addr.Host == "" {
		return fmt.Errorf("storeHost is missing")
	}
	if s.Addr.Port == 0 {
		return fmt.Errorf("storePort is missing")
	}
	if s.Password == "" {
		return fmt.Errorf("logLevel is missing")
	}
	return nil
}
