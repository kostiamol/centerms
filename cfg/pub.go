package cfg

import (
	"fmt"
)

// Pubisher holds publisher's configuration.
type Pubisher struct {
	Addr          Addr
	CfgPatchTopic string
}

func (p Pubisher) validate() error {
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
