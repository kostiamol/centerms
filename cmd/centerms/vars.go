package main

import (
	"flag"
	"os"
	"time"

	"github.com/kostiamol/centerms/svc"

	"github.com/Sirupsen/logrus"
)

// Query the Consul for services:
// dig +noall +answer @127.0.0.1 -p 8600 myCoolServiceName.service.dc1.consul
// curl localhost:8500/v1/health/service/myCoolServiceName?passing

const (
	host          = "127.0.0.1"
	meshAgentName = "centerms"

	defaultWebPort    = 3301
	defaultStreamPort = 3546
	defaultRedisPort  = 6379

	defaultStorePort = defaultRedisPort
	defaultStoreHost = host

	defaultRetryInterval = time.Second * 10
	defaultTTLInterval   = time.Second * 4

	devCfgChan  = "dev_cfg"
	devDataChan = "dev_data"
)

var (
	flagProd                = flag.Bool("prod", false, "If true, HTTPS server will be started")
	flagRedirectHTTPToHTTPS = flag.Bool("redirect-to-https", false, "If true, HTTP will be redirected to HTTPS")

	log = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Level:     logrus.DebugLevel,
	}

	ctrl = svc.Ctrl{StopChan: make(chan struct{})}

	storeHost = flag.String("store-addr", defaultStoreHost, "Store IP address")
	storePort = flag.Int("store-port", defaultStorePort, "Store TCP port")
	storeAddr = svc.Addr{
		Host: *storeHost,
		Port: *storePort,
	}

	rpcPort    = flag.Int("rpc-port", defaultWebPort, "Port to listen on web clients")
	restPort   = flag.Int("rest-port", defaultWebPort, "Port to listen on web clients")
	streamPort = flag.Int("stream-port", defaultStreamPort, "Port for data streaming")

	retry = flag.Duration("retry", defaultRetryInterval, "Retry interval")
	ttl   = flag.Duration("ttl", defaultTTLInterval, "Service TTL check duration")
)
