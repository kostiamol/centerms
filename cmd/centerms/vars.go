package main

import (
	"flag"
	s "github.com/kostiamol/centerms/store/redis"
	"github.com/kostiamol/centerms/svc"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entity"
)

const (
	localhost = "127.0.0.1"
	webHost   = "0.0.0.0"

	// service ports
	defaultDevCfgPort  = 3092
	defaultDevDataPort = 3126
	defaultWebPort     = 3301
	defaultStreamPort  = 3546

	// store vendor ports
	defaultRedisPort      = 6379
	defaultInfluxDBPort   = 8086
	defaultPostgreSQLPort = 5432

	// store address
	defaultStorePort = defaultRedisPort
	defaultStoreHost = localhost

	defaultRetryInterval = time.Second * 10
	defaultTTLInterval   = time.Second * 4

	// consul agent names
	cfgAgentName    = "center_cfg"
	dataAgentName   = "center_data"
	webAgentName    = "center_web"
	streamAgentName = "center_stream"

	redisAgentName = "redis"
	storeAgentName = redisAgentName
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
	storeAddr = entity.Addr{
		Host: *storeHost,
		Port: *storePort,
	}
	store = s.NewRedis(storeAddr, logrus.NewEntry(log), *retry, storeAgentName, *ttl)

	devCfgPort  = flag.Int("dev-cfg-port", defaultDevCfgPort, "Port to listen on configuration from devices")
	devDataPort = flag.Int("dev-data-port", defaultDevDataPort, "Port to listen on data from devices")
	webPort     = flag.Int("web-port", defaultWebPort, "Port to listen on web clients")
	streamPort  = flag.Int("stream-port", defaultStreamPort, "Port for data streaming")

	retry = flag.Duration("retry", defaultRetryInterval, "Retry interval")
	ttl   = flag.Duration("ttl", defaultTTLInterval, "Service TTL check duration")
)
