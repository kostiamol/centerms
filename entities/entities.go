package entities

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type RoutinesController struct {
	StopChan chan struct{}
}

type Server struct {
	Host string
	Port uint
}

type ServerError struct {
	Error   error
	Message string
	Code    int
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

type Request struct {
	Action string          `json:"action"`
	Time   int64           `json:"time"`
	Meta   DevMeta         `json:"meta"`
	Data   json.RawMessage `json:"data"`
}

type DevData struct {
	Site string              `json:"site"`
	Meta DevMeta             `json:"meta"`
	Data map[string][]string `json:"data"`
}

type ConnectionPool struct {
	sync.Mutex
	conn map[string]net.Conn
}

func (c *RoutinesController) Close() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}

func (c *RoutinesController) Wait() {
	<-c.StopChan
	<-time.NewTimer(5).C
}

func (pool *ConnectionPool) AddConn(conn net.Conn, key string) {
	pool.Lock()
	pool.conn[key] = conn
	defer pool.Unlock()
}

func (pool *ConnectionPool) GetConn(key string) net.Conn {
	pool.Lock()
	defer pool.Unlock()
	return pool.conn[key]
}

func (pool *ConnectionPool) RemoveConn(key string) {
	pool.Lock()
	defer pool.Unlock()
	delete(pool.conn, key)
}

func (pool *ConnectionPool) Init() {
	pool.Lock()
	defer pool.Unlock()
	pool.conn = make(map[string]net.Conn)
}
