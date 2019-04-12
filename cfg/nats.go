package cfg

import (
	"fmt"
)

// NATS holds nats configuration.
type NATS struct {
	Host string
	Port uint64
}

func (n NATS) validate() error {
	if n.Host == "" {
		return fmt.Errorf("natsHost is missing")
	}
	if n.Port == 0 {
		return fmt.Errorf("natsPort is missing")
	}
	return nil
}
