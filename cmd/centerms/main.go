package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
)

// todo: add consul agent to center
// todo: reconnect + conn pool
// todo: add Prometheus
// todo: update README.md

func main() {
	flag.Parse()
	checkCLIArgs()

	center := services.NewCenterService()
	center.Run()
	ctrl.Wait()
	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": entities.EventMSTerminated,
	}).Info("center is down")
}
