package cfg

import (
	"fmt"
	"time"
)

// Service holds basic service configuration.
type Service struct {
	AppID                     string
	LogLevel                  string
	RetryNumber               uint64
	RetryTimeout              time.Duration
	PortRPC                   uint64
	PortREST                  uint64
	PortWebSocket             uint64
	RoutineTerminationTimeout time.Duration
}

func (s Service) validate() error {
	if s.AppID == "" {
		return fmt.Errorf("AppID is missing")
	}
	if s.LogLevel == "" {
		return fmt.Errorf("LogLevel is missing")
	}
	if s.RetryNumber == 0 {
		return fmt.Errorf("RetryNumber is missing")
	}
	if s.RetryTimeout == 0 {
		return fmt.Errorf("RetryTimeout is missing")
	}
	if s.PortRPC == 0 {
		return fmt.Errorf("PortRPC is missing")
	}
	if s.PortREST == 0 {
		return fmt.Errorf("PortREST is missing")
	}
	if s.PortWebSocket == 0 {
		return fmt.Errorf("PortWebSocket is missing")
	}
	if s.RoutineTerminationTimeout == 0 {
		return fmt.Errorf("RoutineTerminationTimeout is missing")
	}
	return nil
}
