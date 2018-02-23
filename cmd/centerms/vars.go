package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/centerms/entities"
)

const (
	localhost = "127.0.0.1"
	webHost   = "0.0.0.0"

	// ms ports
	defaultDevConfigPort = "3092"
	defaultDevDataPort   = "3126"
	defaultWebPort       = "3301"
	defaultStreamPort    = "3546"

	// storage ports
	defaultRedisPort      = "6379"
	defaultInfluxDBPort   = "8086"
	defaultPostgreSQLPort = "5432"

	defaultStoragePort = defaultRedisPort
	defaultStorageHost = "127.0.0.1"

	defaultRetryInterval = time.Second * 10
	defaultTTLInterval   = time.Second * 4
)

var (
	log = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Level:     logrus.DebugLevel,
	}

	ctrl = entities.ServiceController{StopChan: make(chan struct{})}

	storageHost = flag.String("storage-addr", defaultStorageHost, "Storage address")
	storagePort = flag.String("storage-port", defaultStoragePort, "Port to listen on dat from storage")

	devConfigPort = flag.String("dev-config-port", defaultDevConfigPort, "Port to listen on config from devices")
	devDataPort   = flag.String("dev-data-port", defaultDevDataPort, "Port to listen on data from devices")
	webPort       = flag.String("web-port", defaultWebPort, "Port to listen on clients")
	streamPort    = flag.String("stream-port", defaultStreamPort, "Port for data streaming")

	ttl   = flag.Duration("ttl", defaultTTLInterval, "Service TTL check duration")
	retry = flag.Duration("retry", defaultRetryInterval, "Retry interval")
)

// checkCLIArgs checks whether the args were passed.
func checkCLIArgs() {
	if len(*storageHost) == 0 {
		fmt.Fprintln(os.Stderr, "storage-addr argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if len(*storagePort) == 0 {
		fmt.Fprintln(os.Stderr, "storage-port argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if len(*devConfigPort) == 0 {
		fmt.Fprintln(os.Stderr, "dev-config-port argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if len(*devDataPort) == 0 {
		fmt.Fprintln(os.Stderr, "dev-data-port argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if len(*webPort) == 0 {
		fmt.Fprintln(os.Stderr, "web-port argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if len(*streamPort) == 0 {
		fmt.Fprintln(os.Stderr, "stream-port argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if *ttl == 0 {
		fmt.Fprintln(os.Stderr, "ttl argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	} else if *retry == 0 {
		fmt.Fprintln(os.Stderr, "retry argument is required")
		flag.PrintDefaults()
		os.Exit(1)
	}
}
