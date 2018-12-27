// Package store provides means for data storage and retrieving.
package store

import (
	"strings"

	"github.com/garyburd/redigo/redis"

	"fmt"

	"github.com/kostiamol/centerms/svc"
	"github.com/pkg/errors"
)

const (
	partialDevKey       = "device:"
	partialDevCfgKey    = ":cfg"
	partialDevParamsKey = ":params"
)

// Redis is used to provide a storage based on Redis according to Storer interface.
type Redis struct {
	addr svc.Addr
	conn redis.Conn
}

// NewRedis creates a new instance of Redis store.
func NewRedis(a svc.Addr) svc.Storer {
	return &Redis{
		addr: a,
	}
}

// Init initializes an instance of Redis.
func (r *Redis) Init() error {
	if r.addr.Host == "" {
		return errors.New("store: Init(): host is empty")
	} else if r.addr.Port == 0 {
		return errors.New("store: Init(): port is empty")
	}
	var err error
	r.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	if err != nil {
		return errors.Wrap(err, "store: Init(): Dial() failed: ")
	}
	return nil
}

// CreateConn creates, initializes and returns a new instance of Redis.
func (r *Redis) CreateConn() (svc.Storer, error) {
	store := Redis{
		addr: r.addr,
	}
	var err error
	store.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	if err != nil {
		return nil, errors.Wrap(err, "store: CreateConn(): Dial() failed: ")
	}
	return &store, nil
}

// CloseConn closes connection with Redis.
func (r *Redis) CloseConn() error {
	return r.conn.Close()
}

// GetDevsData returns the data concerning all the devices.
func (r *Redis) GetDevsData() ([]svc.DevData, error) {
	devParamsKeys, err := redis.Strings(r.conn.Do("SMEMBERS", "devParamsKeys"))
	if err != nil {
		errors.Wrap(err, "store: GetDevsData(): SMEMBERS for devParamsKeys failed: ")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for k, v := range devParamsKeys {
		devParamsKeysTokens[k] = strings.Split(v, ":")
	}

	var (
		devData  svc.DevData
		devsData []svc.DevData
	)

	for index, key := range devParamsKeysTokens {
		devData.Meta = svc.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		data := make(map[string][]string)
		params, err := redis.Strings(r.conn.Do("SMEMBERS", devParamsKeys[index]))
		if err != nil {
			errors.Wrapf(err, "store: GetDevsData(): SMEMBERS() for %s failed: ", devParamsKeys[index])
		}

		for i, p := range params {
			data[p], err = redis.Strings(r.conn.Do("ZRANGEBYSCORE", devParamsKeys[index]+":"+p, "-inf", "inf"))
			if err != nil {
				errors.Wrapf(err, "store: GetDevsData(): ZRANGEBYSCORE() for %s failed: ", devParamsKeys[i])
			}
		}
		devsData = append(devsData, devData)
	}
	return devsData, err
}

// GetDevMeta returns metadata for the given device.
func (r *Redis) GetDevMeta(id svc.DevID) (*svc.DevMeta, error) {
	t, err := redis.String(r.conn.Do("HGET", "id:"+id, "type"))
	if err != nil {
		errors.Wrap(err, "store: GetDevMeta(): HGET() failed: ")
	}
	n, err := redis.String(r.conn.Do("HGET", "id:"+id, "name"))
	if err != nil {
		errors.Wrap(err, "store: GetDevMeta():  HGET() failed: ")
	}
	m, err := redis.String(r.conn.Do("HGET", "id:"+id, "mac"))
	if err != nil {
		errors.Wrap(err, "store: GetDevMeta():  HGET() failed: ")
	}

	return &svc.DevMeta{
		Type: t,
		Name: n,
		MAC:  m,
	}, nil
}

// SetDevMeta sets metadata for the given device.
func (r *Redis) SetDevMeta(m *svc.DevMeta) error {
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "type", m.Type); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSet() failed: ")
	}
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "name", m.Name); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSet() failed: ")
	}
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "mac", m.MAC); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSet() failed: ")
	}
	return nil
}

// DevIsRegistered returns 'true' if the given device is registered, otherwise - 'false'.
func (r *Redis) DevIsRegistered(m *svc.DevMeta) (bool, error) {
	cfgKey := m.MAC + partialDevCfgKey
	if ok, err := redis.Bool(r.conn.Do("EXISTS", cfgKey)); ok {
		if err != nil {
			return false, errors.Wrap(err, "store: DevIsRegistered(): EXISTS() failed: ")
		}
		return true, nil
	}
	return false, nil
}

// GetDevData returns data concerning given device.
func (r *Redis) GetDevData(id svc.DevID) (*svc.DevData, error) {
	m, err := r.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch m.Type {
	case "fridge":
		return r.getFridgeData(m)
	case "washer":
		return r.getWasherData(m)
	default:
		return nil, errors.New("store: GetDevData(): dev type is unknown")
	}
}

// SaveDevData saves data concerning given device.
func (r *Redis) SaveDevData(d *svc.DevData) error {
	switch d.Meta.Type {
	case "fridge":
		return r.saveFridgeData(d)
	case "washer":
		return r.saveWasherData(d)
	default:
		return errors.New("store: SaveDevData(): dev type is unknown")
	}
}

// GetDevCfg returns configuration for the given device.
func (r *Redis) GetDevCfg(id svc.DevID) (*svc.DevCfg, error) {
	m, err := r.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch m.Type {
	case "fridge":
		return r.getFridgeCfg(m)
	case "washer":
		return r.getWasherCfg(m)
	default:
		return nil, errors.New("store: GetDevCfg(): dev type is unknown")
	}
}

// SetDevCfg sets configuration for the given device.
func (r *Redis) SetDevCfg(id svc.DevID, c *svc.DevCfg) error {
	m, err := r.GetDevMeta(id)
	if err != nil {
		return err
	}

	switch m.Type {
	case "fridge":
		return r.setFridgeCfg(c, m)
	case "washer":
		return r.setWasherCfg(c)
	default:
		return errors.New("store: SetDevCfg(): dev type is unknown")
	}
}

// GetDevDefaultCfg returns default configuration for the given device.
func (r *Redis) GetDevDefaultCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	switch m.Type {
	case "fridge":
		return r.getFridgeDefaultCfg(m)
	case "washer":
		return r.getWasherDefaultCfg(m)
	default:
		return nil, errors.New("store: GetDevDefaultCfg(): dev type is unknown")
	}
}

// Publish posts a message on the given channel.
func (r *Redis) Publish(msg interface{}, channel string) (int64, error) {
	numberOfClients, err := redis.Int64(r.conn.Do("PUBLISH", msg, channel))
	if err != nil {
		return 0, errors.Wrap(err, "store: Publish() failed: ")
	}
	return numberOfClients, nil
}

// Subscribe subscribes the client to the specified channels.
func (r *Redis) Subscribe(c chan []byte, channel ...string) {
	conn := redis.PubSubConn{Conn: r.conn}
	for _, cn := range channel {
		conn.Subscribe(cn)
	}
	for {
		switch v := conn.Receive().(type) {
		case redis.Message:
			c <- v.Data
		}
	}
}
