package cfg

import (
	"fmt"
	"time"
)

// Service holds basic service configuration.
type Service struct {
	AppID              string
	LogLevel           string
	RetryTimeout       time.Duration
	RetryAttempts      uint32
	PortRPC            uint32
	PortREST           uint32
	PortWebSocket      uint32
	TerminationTimeout time.Duration
}

func (s Service) validate() error {
	if s.AppID == "" {
		return fmt.Errorf("app id env var is missing")
	}
	if s.LogLevel == "" {
		return fmt.Errorf("log level env var is missing")
	}
	if s.RetryAttempts == 0 {
		return fmt.Errorf("retry attempts env var is missing")
	}
	if s.RetryTimeout == 0 {
		return fmt.Errorf("retry timeout env var is missing")
	}
	if s.PortRPC == 0 {
		return fmt.Errorf("rpc port env var is missing")
	}
	if s.PortREST == 0 {
		return fmt.Errorf("rest port env var is missing")
	}
	if s.PortWebSocket == 0 {
		return fmt.Errorf("websocket port env var is missing")
	}
	if s.TerminationTimeout == 0 {
		return fmt.Errorf("termination timeout env var is missing")
	}
	return nil
}
