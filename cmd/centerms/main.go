package main

import (
	"os"

	"github.com/kostiamol/centerms/cfg"
	s "github.com/kostiamol/centerms/store"

	"github.com/kostiamol/centerms/api"

	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/svc"
)

// todo: add Prometheus
// todo: update README.md
// todo: reconnect + conn pool

func main() {
	flag.Parse()

	m := svc.NewMeshAgent(
		&svc.MeshAgentCfg{
			Name: meshAgentName,
			Port: defaultWebPort,
			TTL:  *ttl,
			Log:  logrus.NewEntry(log),
		})
	go m.Run()

	store := s.NewRedis(storeAddr)
	if err := store.Init(); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":  "main",
			"event": cfg.EventStoreInit,
		}).Errorf("Init() failed: %s", err)
		os.Exit(1)
	}

	d := svc.NewDataService(
		&svc.DataServiceCfg{
			Store:   store,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			PubChan: devDataChan,
		})
	go d.Run()

	s := svc.NewStreamService(
		&svc.StreamServiceCfg{
			Addr: svc.Addr{
				Host: host,
				Port: *streamPort,
			},
			Store:   store,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			PubChan: devDataChan,
		})
	go s.Run()

	c := svc.NewCfgService(
		&svc.CfgServiceCfg{
			Store:   store,
			Ctrl:    ctrl,
			Log:     logrus.NewEntry(log),
			Retry:   *retry,
			SubChan: devCfgChan,
		})
	go c.Run()

	a := api.NewAPI(
		&api.APICfg{
			Host:        host,
			RPCPort:     *rpcPort,
			RESTPort:    *restPort,
			CfgProvider: c,
			Log:         logrus.NewEntry(log),
			PubChan:     devCfgChan,
		})
	go a.Run()

	ctrl.Wait()

	logrus.WithFields(logrus.Fields{
		"func":  "main",
		"event": cfg.EventMSTerminated,
	}).Info("centerms is down")
}
