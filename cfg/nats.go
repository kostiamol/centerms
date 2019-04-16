package cfg

import (
	"fmt"
)

// NATS holds nats configuration.
type NATS struct {
	Addr Addr
}

func (n NATS) validate() error {
	if n.Addr.Host == "" {
		return fmt.Errorf("natsHost is missing")
	}
	if n.Addr.Port == 0 {
		return fmt.Errorf("natsPort is missing")
	}
	return nil
}
