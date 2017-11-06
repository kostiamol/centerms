package main

import (
	"os"
	"time"
)

const (
	localhost            = "127.0.0.1"
	webHost              = "0.0.0.0"

	defaultDevConfigPort = "3092"
	defaultDevDataPort   = "3126"
	defaultWebPort       = "3301"
	defaultStreamPort    = "3546"

	defaultRedisPort      = "6379"
	defaultPostgreSQLPort = "5432"
	defaultStoragePort    = defaultRedisPort
	defaultStorageHost    = "127.0.0.1"

	retryInterval           = time.Second * 10
)

var (
	storageHost   = getEnvVar("STORAGE_TCP_ADDR", defaultStorageHost)
	storagePort   = getEnvVar("STORAGE_TCP_PORT", defaultStoragePort)
	devConfigPort = getEnvVar("CENTER_CONFIG_TCP_PORT", defaultDevConfigPort)
	devDataPort   = getEnvVar("CENTER_DATA_TCP_PORT", defaultDevDataPort)
	webPort       = getEnvVar("WEB_TCP_PORT", defaultWebPort)
	streamPort    = getEnvVar("STREAM_TCP_PORT", defaultStreamPort)
)

// getEnvVar checks whether environmental variable with name 'key' was specified.
// It returns that variable if it was set and defaultVal otherwise.
func getEnvVar(key string, defaultVal string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		return defaultVal
	}
	return val
}
