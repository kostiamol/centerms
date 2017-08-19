package devices

import (
	"github.com/giperboloid/centerms/entities"
	"github.com/giperboloid/centerms/db"

	"net/http"
	"net"
)

type DevConfigDriver interface {
	GetDevConfig(configInfo, mac string, client db.Client) (*entities.DevConfig)
	SetDevConfig(configInfo string, config *entities.DevConfig, client db.Client)
	GetDefaultConfig() (*entities.DevConfig)
}

type DevDataDriver interface {
	GetDevData(devParamsKey string, devMeta entities.DevMeta, client db.Client) entities.DevData
	SetDevData(req *entities.Request, worker db.Client) *entities.ServerError
}
type DevServerHandler interface {
	SendDefaultConfigurationTCP(conn net.Conn, dbClient db.Client, req *entities.Request) ([]byte)
	PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta entities.DevMeta, client db.Client)
}
type Driver interface {
	DevDataDriver
	DevServerHandler
	DevConfigDriver
}
