package entities

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type DevStore interface {
	DevDataStore
	DevConfigStore
	// substitute with NATS in separate entity
	Notifier
}

type DevDataStore interface {
	Store
	GetDevsData() ([]DevData, error)
	GetDevData(m *DevMeta) (*DevData, error)
	SetDevData(r *Request) error
}

type DevConfigStore interface {
	Store
	GetDevConfig(m *DevMeta) (*DevConfig, error)
	SetDevConfig(m *DevMeta, c *DevConfig) error
	GetDevDefaultConfig(m *DevMeta) (*DevConfig, error)
}

type Store interface {
	SetServer(s *Server) error
	CreateConn() (DevStore, error)
	CloseConn() error
}

type Notifier interface {
	Publish(channel string, msg interface{}) (int64, error)
	Subscribe(c chan []string, channel ...string) error
}

type Device interface {
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
