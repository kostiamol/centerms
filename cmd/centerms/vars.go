package main

import (
	"github.com/giperboloid/centerms/entities"
	"os"
	"strconv"
)

var (
	dbServer = entities.Server{
		Host: getEnvDbHost("REDIS_PORT_6379_TCP_ADDR"),
		Port: getEnvDbPort("REDIS_PORT_6379_TCP_PORT"),
	}

	localhost     = "0.0.0.0"
	wsPort        = uint(2540)
	tcpDataPort   = uint(3030)
	tcpConfigPort = uint(3000)
	httpPort      = uint(8100)
)

func getEnvDbPort(key string) uint {
	parsed64, _ := strconv.ParseUint(os.Getenv(key), 10, 64)
	port := uint(parsed64)
	if port == 0 {
		return uint(6379)
	}
	return port
}

func getEnvDbHost(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "127.0.0.1"
	}
	return host
}
