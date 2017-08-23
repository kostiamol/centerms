package entities

import (
	"github.com/giperboloid/centerms/entities"

	"net"
	"net/http"
)

type DevConfigDriver interface {
	GetDevConfig(configInfo, mac string, client entities.Storage) *entities.DevConfig
	SetDevConfig(configInfo string, config *entities.DevConfig, client entities.Storage)
	GetDefaultConfig() *entities.DevConfig
}

type DevDataDriver interface {
	GetDevData(devParamsKey string, devMeta entities.DevMeta, client entities.Storage) entities.DevData
	SetDevData(req *entities.Request, worker entities.Storage) error
}

type DevServerHandler interface {
	SendDefaultConfigurationTCP(conn net.Conn, dbClient entities.Storage, req *entities.Request) []byte
	PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta entities.DevMeta, client entities.Storage)
}

type Driver interface {
	DevDataDriver
	DevServerHandler
	DevConfigDriver
}
