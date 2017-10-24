package main

import (
	"os"
)

const (
	defaultDevConfigPort = "3092"
	defaultDevDataPort   = "3126"
	defaultWebPort       = "3301"
	defaultStreamPort    = "3546"

	defaultStoragePort    = defaultRedisPort
	defaultRedisPort      = "6379"
	defaultPostgreSQLPort = "5432"

	defaultStorageHost = "127.0.0.1"
	localhost          = "0.0.0.0"
)

var (
	storageHost   = getEnvStorageHost("STORAGE_TCP_ADDR")
	storagePort   = getEnvStoragePort("STORAGE_TCP_PORT")
	devConfigPort = getEnvConfigPort("DEV_CONFIG_TCP_PORT")
	devDataPort   = getEnvDataPort("DEV_DATA_TCP_PORT")
	webPort       = getEnvWebPort("DEV_WEB_TCP_PORT")
	streamPort    = getEnvStreamPort("DEV_STREAM_TCP_PORT")
)

func getEnvStorageHost(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return defaultStorageHost
	}
	return host
}

func getEnvStoragePort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return defaultStoragePort
	}
	return port
}

func getEnvConfigPort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return defaultDevConfigPort
	}
	return port
}

func getEnvDataPort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return defaultDevDataPort
	}
	return port
}

func getEnvWebPort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return defaultWebPort
	}
	return port
}

func getEnvStreamPort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return defaultStreamPort
	}
	return port
}
