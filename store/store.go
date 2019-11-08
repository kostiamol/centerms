// Package store provides means for data storage and retrieving.
package store

import (
	"time"

	"github.com/kostiamol/centerms/cfg"

	"github.com/garyburd/redigo/redis"

	"fmt"
)

const (
	partialDevKey       = "device:"
	partialDevCfgKey    = ":cfg"
	partialDevParamsKey = ":params"
)

type (
	// store is used to provide a storage based on redis db under the hood.
	store struct {
		addr cfg.Addr
		pool *redis.Pool
	}

	// Cfg is used to initialize an instance of store.
	Cfg struct {
		Addr             cfg.Addr
		Password         string
		MaxIdlePoolConns uint32
		IdleTimeout      time.Duration
	}
)

// New creates a new instance of store.
func New(c *Cfg) (*store, error) { // nolint
	r := &store{
		addr: c.Addr,
		pool: &redis.Pool{
			MaxIdle:     int(c.MaxIdlePoolConns),
			IdleTimeout: c.IdleTimeout,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", c.Addr.Host, c.Addr.Port))
				if err != nil {
					return nil, fmt.Errorf("func Dial: %s", err)
				}
				//if _, err := c.Do("AUTH", password); err != nil {
				//	c.Close()
				//	return nil, err
				//}
				return c, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if _, err := c.Do("PING"); err != nil {
					return fmt.Errorf("func PING: %s", err)
				}
				return nil
			},
		},
	}

	if _, err := r.ping(); err != nil {
		return nil, fmt.Errorf("func PING: %s", err)
	}
	return r, nil
}

// Close releases the resources used by the pool.
func (s *store) Close() error {
	return s.pool.Close()
}

func (s *store) ping() (string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("PING"))
}

func (s *store) exists(key string) (bool, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Bool(redis.Bytes(conn.Do("EXISTS", key)))
}

func (s *store) multi() (string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.String(redis.Bytes(conn.Do("MULTI")))
}

func (s *store) discard() (string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.String(redis.Bytes(conn.Do("DISCARD")))
}

func (s *store) exec() ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(redis.Bytes(conn.Do("EXEC")))
}

func (s *store) sadd(key string, members ...interface{}) (int, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Int(redis.Bytes(conn.Do("SADD", key, members)))
}

func (s *store) zadd(key string, args ...interface{}) ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(redis.Bytes(conn.Do("ZADD", key, args)))
}

func (s *store) hget(key, field string) (string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.String(redis.Bytes(conn.Do("HGET", key, field)))
}

func (s *store) hmset(key string, fields ...interface{}) ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(redis.Bytes(conn.Do("HMSET", key, fields)))
}

func (s *store) hmget(key string, fields ...interface{}) ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(redis.Bytes(conn.Do("HMGET", key, fields)))
}

func (s *store) smembers(key string) ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", key))
}

func (s *store) zrangebyscore(key string, min, max interface{}) ([]string, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("ZRANGEBYSCORE", key, min, max))
}

func (s *store) publish(msg interface{}, channel string) (int, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return redis.Int(redis.Bytes(conn.Do("PUBLISH", msg, channel)))
}
