// Package store provides means for data storage and retrieving.
package store

import (
	"math/rand"
	"strings"

	"github.com/kostiamol/centerms/cfg"

	"time"

	"github.com/garyburd/redigo/redis"

	"fmt"

	"github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
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
	addr      svc.Addr
	conn      redis.Conn
	log       *logrus.Entry
	retry     time.Duration
	agentName string
	agent     *consul.Agent
	ttl       time.Duration
}

// NewRedis creates a new instance of Redis store.
func NewRedis(a svc.Addr, l *logrus.Entry, retry time.Duration, agentName string,
	ttl time.Duration) svc.Storer {

	return &Redis{
		addr:      a,
		agentName: agentName,
		ttl:       ttl,
		retry:     retry,
		log:       l.WithFields(logrus.Fields{"component": "store"}),
	}
}

// Init initializes an instance of Redis.
func (r *Redis) Init() error {
	if r.addr.Host == "" {
		return errors.New("Redis: SetServer(): host is empty")
	} else if r.addr.Port == 0 {
		return errors.New("Redis: SetServer(): port is empty")
	}

	var err error
	r.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	for err != nil {
		r.log.WithFields(logrus.Fields{
			"func": "Run",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(r.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		r.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	}

	if ok, err := r.Check(); !ok {
		return err
	}

	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return errors.Errorf("consul: %s", err)
	}
	r.agent = c.Agent()
	agent := &consul.AgentServiceRegistration{
		Name: r.agentName,
		Check: &consul.AgentServiceCheck{
			TTL: r.ttl.String(),
		},
	}
	if err := r.agent.ServiceRegister(agent); err != nil {
		return errors.Errorf("consul: %s", err)
	}
	go r.UpdateTTL(r.Check)

	return nil
}

// Check issues PING Redis command to check if Redis is ok.
func (r *Redis) Check() (bool, error) {
	if _, err := r.conn.Do("PING"); err != nil {
		return false, err
	}
	return true, nil
}

// UpdateTTL updates TTL (Time to live).
func (r *Redis) UpdateTTL(check func() (bool, error)) {
	t := time.NewTicker(r.ttl / 2)
	for range t.C {
		r.update(check)
	}
}

func (r *Redis) update(check func() (bool, error)) {
	var health string
	if ok, err := check(); !ok {
		r.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data.
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := r.agent.UpdateTTL("svc:"+r.agentName, "", health); err != nil {
		r.log.WithFields(logrus.Fields{
			"func":  "update",
			"event": cfg.EventUpdConsulStatus,
		}).Error(err)
	}
}

// CreateConn creates, initializes and returns a new instance of Redis.
func (r *Redis) CreateConn() (svc.Storer, error) {
	store := Redis{
		addr:  r.addr,
		retry: r.retry,
		log:   r.log,
	}

	var err error
	store.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	for err != nil {
		r.log.WithFields(logrus.Fields{
			"func": "Run",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(r.retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		store.conn, err = redis.Dial("tcp", r.addr.Host+":"+fmt.Sprint(r.addr.Port))
	}
	return &store, err
}

// CloseConn closes connection with Redis.
func (r *Redis) CloseConn() error {
	return r.conn.Close()
}

// GetDevsData returns the data concerning all the devices.
func (r *Redis) GetDevsData() ([]svc.DevData, error) {
	devParamsKeys, err := redis.Strings(r.conn.Do("SMEMBERS", "devParamsKeys"))
	if err != nil {
		errors.Wrap(err, "Redis: GetDevsData(): SMembers for devParamsKeys has failed")
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
			errors.Wrapf(err, "Redis: GetDevsData(): SMembers() for %s has failed", devParamsKeys[index])
		}

		for i, p := range params {
			data[p], err = redis.Strings(r.conn.Do("ZRANGEBYSCORE", devParamsKeys[index]+":"+p, "-inf", "inf"))
			if err != nil {
				errors.Wrapf(err, "Redis: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
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
		errors.Wrapf(err, "Redis: GetDevMeta(): HGet() has failed")
	}
	n, err := redis.String(r.conn.Do("HGET", "id:"+id, "name"))
	if err != nil {
		errors.Wrapf(err, "Redis: GetDevMeta():  HGet() has failed")
	}
	m, err := redis.String(r.conn.Do("HGET", "id:"+id, "mac"))
	if err != nil {
		errors.Wrapf(err, "Redis: GetDevMeta():  HGet() has failed")
	}

	meta := svc.DevMeta{
		Type: t,
		Name: n,
		MAC:  m,
	}
	return &meta, nil
}

// SetDevMeta sets metadata for the given device.
func (r *Redis) SetDevMeta(m *svc.DevMeta) error {
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "type", m.Type); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		return err
	}
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "name", m.Name); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		return err
	}
	if _, err := r.conn.Do("HMSET", "id:"+m.MAC, "mac", m.MAC); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		return err
	}
	return nil
}

// DevIsRegistered returns 'true' if the given device is registered, otherwise - 'false'.
func (r *Redis) DevIsRegistered(m *svc.DevMeta) (bool, error) {
	cfgKey := m.MAC + partialDevCfgKey
	if ok, err := redis.Bool(r.conn.Do("EXISTS", cfgKey)); ok {
		if err != nil {
			errors.Wrap(err, "Redis: DevIsRegistered(): Exists() has failed")
		}
		return true, err
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
		return &svc.DevData{}, errors.New("Redis: GetDevData(): dev type is unknown")
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
		return errors.New("Redis: SaveDevData(): dev type is unknown")
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
		return &svc.DevCfg{}, errors.New("Redis: GetDevCfg(): dev type is unknown")
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
		return errors.New("Redis: SetDevCfg(): dev type is unknown")
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
		return &svc.DevCfg{}, errors.New("Redis: GetDevDefaultCfg(): dev type is unknown")
	}
}

// Publish posts a message on the given channel.
func (r *Redis) Publish(msg interface{}, channel string) (int64, error) {
	numberOfClients, err := redis.Int64(r.conn.Do("PUBLISH", msg, channel))
	if err != nil {
		return 0, err
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
			r.log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Infof("chan %s: message: %s", v.Channel, v.Data)
			c <- v.Data
		case redis.Subscription:
			r.log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Infof("chan %s: kind: %s count: %d", v.Channel, v.Kind, v.Count)
		case error:
			r.log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Errorf("%s", v)
		}
	}
}
