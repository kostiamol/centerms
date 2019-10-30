package cfg

import (
	"fmt"
)

// Publisher holds publisher's configuration.
type Publisher struct {
	Addr          Addr
	CfgPatchTopic string
}

func (p Publisher) validate() error {
	if p.Addr.Host == "" {
		return fmt.Errorf("publisher host env var is missing")
	}
	if p.Addr.Port == 0 {
		return fmt.Errorf("publisher port env var is missing")
	}
	if p.CfgPatchTopic == "" {
		return fmt.Errorf("publisher cfg patch topic env var is missing")
	}
	return nil
}
