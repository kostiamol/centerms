package entities

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type Storage interface {
	SetServer(s *Server) error

	CreateConnection() (Storage, error)
	CloseConnection() error

	GetDevsData() ([]DevData, error)
	GetDevData(devParamsKey string, m *DevMeta) (*DevData, error)
	SetDevData(req *Request) error

	GetConfigKey(mac string) (string, error)
	GetDevConfig(t string, configInfo string, mac string) (*DevConfig, error)
	SetDevConfig(t string, configInfo string, c *DevConfig) error
	GetDevDefaultConfig(t string, m *DevMeta) (*DevConfig, error)
	SendDevDefaultConfig(c *net.Conn, req *Request) ([]byte, error)

	Publish(channel string, msg interface{}) (int64, error)
	Subscribe(c chan []string, channel ...string) error
}

type Notifier interface {

}

type Device interface {

}

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
	PatchDevConfigHandlerHTTP()
}

type Driver interface {
	DevDataDriver
	DevServerHandler
	DevConfigDriver
}

type RoutinesController struct {
	StopChan chan struct{}
}

type Server struct {
	Host string
	Port uint
}

type Request struct {
	Action string          `json:"action"`
	Time   int64           `json:"time"`
	Meta   DevMeta         `json:"meta"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	Status int    `json:"status"`
	Descr  string `json:"descr"`
}

type DevConfig struct {
	MAC  string          `json:"mac"`
	Data json.RawMessage `json:"data"`
}

type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IP   string `json:"ip"`
}

type DevData struct {
	Site string              `json:"site"`
	Meta DevMeta             `json:"meta"`
	Data map[string][]string `json:"data"`
}

type ConnPool struct {
	sync.Mutex
	conn map[string]net.Conn
}

func (pool *ConnPool) Init() {
	pool.Lock()
	defer pool.Unlock()
	pool.conn = make(map[string]net.Conn)
}

func (c *RoutinesController) Wait() {
	<-c.StopChan
	<-time.NewTimer(5).C
}

func (pool *ConnPool) AddConn(conn net.Conn, key string) {
	pool.Lock()
	pool.conn[key] = conn
	defer pool.Unlock()
}

func (pool *ConnPool) GetConn(key string) net.Conn {
	pool.Lock()
	defer pool.Unlock()
	return pool.conn[key]
}

func (pool *ConnPool) RemoveConn(key string) {
	pool.Lock()
	defer pool.Unlock()
	delete(pool.conn, key)
}

func (c *RoutinesController) Close() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
