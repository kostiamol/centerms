package cfg

import "fmt"

// TraceAgent holds trace agent configuration.
type TraceAgent struct {
	Addr Addr
}

func (t TraceAgent) Validate() error {
	if t.Addr.Host == "" {
		return fmt.Errorf("trace agent addr host env var is missing")
	}
	if t.Addr.Port == 0 {
		return fmt.Errorf("trace agent addr port env var is missing")
	}
	return nil
}
