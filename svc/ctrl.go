package svc

import (
	"os"
	"os/signal"
	"time"
)

// Ctrl contains StopChan that allows to terminate all the services that listen the channel.
type Ctrl struct {
	StopChan chan struct{}
}

// Wait waits until StopChan will be closed and then makes a pause for the amount seconds defined in variable
// timeForRoutineTermination in order to give time for all the services to shutdown gracefully.
func (c *Ctrl) Wait(t time.Duration) {
	inter := make(chan os.Signal, 1)
	signal.Notify(inter, os.Interrupt)

	select {
	case <-inter:
		close(c.StopChan)
	case <-c.StopChan:
	}

	<-time.NewTimer(t).C
}

// Terminate closes StopChan to signal all the services to shutdown.
func (c *Ctrl) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
