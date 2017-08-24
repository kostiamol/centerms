package entities

import (

	"net"
	"net/http"
)

type DevConfigDriver interface {
	GetDevConfig(configInfo, mac string, client Storage) *DevConfig
	SetDevConfig(configInfo string, config *DevConfig, client Storage)
	GetDefaultConfig() *DevConfig
}

type DevDataDriver interface {
	GetDevData(devParamsKey string, devMeta DevMeta, client Storage) DevData
	SetDevData(req *Request, worker Storage) error
}

type DevServerHandler interface {
	SendDefaultConfigurationTCP(conn net.Conn, dbClient Storage, req *Request) []byte
	PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta DevMeta, client Storage)
}

type Driver interface {
	DevDataDriver
	DevServerHandler
	DevConfigDriver
}
