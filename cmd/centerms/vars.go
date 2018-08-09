package main

import (
	"flag"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/storages/redis"
)

const (
	localhost = "127.0.0.1"
	webHost   = "0.0.0.0"

	// service ports
	defaultDevConfigPort = 3092
	defaultDevDataPort   = 3126
	defaultWebPort       = 3301
	defaultStreamPort    = 3546

	// storage vendor ports
	defaultRedisPort      = 6379
	defaultInfluxDBPort   = 8086
	defaultPostgreSQLPort = 5432

	// storage address
	defaultStoragePort = defaultRedisPort
	defaultStorageHost = localhost

	defaultRetryInterval = time.Second * 10
	defaultTTLInterval   = time.Second * 4

	// consul agent names
	configAgentName = "center_config"
	dataAgentName   = "center_data"
	webAgentName    = "center_web"
	streamAgentName = "center_stream"

	redisAgentName   = "redis"
	storageAgentName = redisAgentName
)

var (
	flagProd                = flag.Bool("prod", false, "If true, HTTPS server will be started")
	flagRedirectHTTPToHTTPS = flag.Bool("redirect-to-https", false, "If true, HTTP will be redirected to HTTPS")

	log = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Level:     logrus.DebugLevel,
	}
	ctrl = entities.ServiceController{StopChan: make(chan struct{})}

	storageHost = flag.String("storage-addr", defaultStorageHost, "Storage IP address")
	storagePort = flag.Int("storage-port", defaultStoragePort, "Storage TCP port")
	storageAddr = entities.Address{
		Host: *storageHost,
		Port: *storagePort,
	}
	storage = storages.NewRedisStorage(storageAddr, logrus.NewEntry(log), *retry, storageAgentName, *ttl)

	devConfigPort = flag.Int("dev-config-port", defaultDevConfigPort, "Port to listen on config from devices")
	devDataPort   = flag.Int("dev-data-port", defaultDevDataPort, "Port to listen on data from devices")
	webPort       = flag.Int("web-port", defaultWebPort, "Port to listen on web clients")
	streamPort    = flag.Int("stream-port", defaultStreamPort, "Port for data streaming")

	retry = flag.Duration("retry", defaultRetryInterval, "Retry interval")
	ttl   = flag.Duration("ttl", defaultTTLInterval, "Service TTL check duration")
)
