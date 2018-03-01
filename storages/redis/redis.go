package storages

import (
	"math/rand"
	"strings"

	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/pkg/errors"
)

const (
	partialDevKey       = "device:"
	partialDevConfigKey = ":config"
	partialDevParamsKey = ":params"
)

type RedisStorage struct {
	entities.ExternalService
	conn  redis.Conn
	Retry time.Duration
	Log   *logrus.Entry
}

func NewRedisStorage(addr entities.Address, name string, ttl time.Duration, retry time.Duration,
	log *logrus.Entry) *RedisStorage {

	return &RedisStorage{
		ExternalService: entities.ExternalService{
			Addr: addr,
			Name: name,
			TTL:  ttl,
		},
		Retry: retry,
		Log:   log.WithFields(logrus.Fields{"service": "storage", "type": "redis"}),
	}
}

func (s *RedisStorage) Init() error {
	if s.Addr.Host == "" {
		return errors.New("RedisStorage: SetServer(): host is empty")
	} else if s.Addr.Port == "" {
		return errors.New("RedisStorage: SetServer(): port is empty")
	}

	var err error
	s.conn, err = redis.Dial("tcp", s.Addr.Host+":"+s.Addr.Port)
	for err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "Init",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.Retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		s.conn, err = redis.Dial("tcp", s.Addr.Host+":"+s.Addr.Port)
	}

	if ok, err := s.Check(); !ok {
		return err
	}

	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return errors.Errorf("Consul: %s", err)
	}
	s.ConsulAgent = c.Agent()

	serviceDef := &consul.AgentServiceRegistration{
		Name: s.Name,
		Check: &consul.AgentServiceCheck{
			TTL: s.TTL.String(),
		},
	}

	if err := s.ConsulAgent.ServiceRegister(serviceDef); err != nil {
		return errors.Errorf("Consul: %s", err)
	}
	go s.UpdateTTL(s.Check)

	return nil
}

// Check method issues PING Redis command to check if we’re ok.
func (s *RedisStorage) Check() (bool, error) {
	_, err := s.conn.Do("PING")
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *RedisStorage) UpdateTTL(check func() (bool, error)) {
	ticker := time.NewTicker(s.TTL / 2)
	for range ticker.C {
		s.update(check)
	}
}

func (s *RedisStorage) update(check func() (bool, error)) {
	var health string
	ok, err := check()
	if !ok {
		s.Log.WithFields(logrus.Fields{
			"func":  "update",
			"event": "updating_storage_status",
		}).Errorf("check has failed: %s", err)

		// failed check will remove a service instance from DNS and HTTP query
		// to avoid returning errors or invalid data
		health = consul.HealthCritical
	} else {
		health = consul.HealthPassing
	}

	if err := s.ConsulAgent.UpdateTTL("service:"+s.Name, "", health); err != nil {
		s.Log.WithFields(logrus.Fields{
			"func":  "update",
			"event": "updating_storage_status",
		}).Error(err)
	}
}

func (s *RedisStorage) CreateConn() (entities.Storager, error) {
	newStorage := RedisStorage{
		ExternalService: entities.ExternalService{Addr: s.Addr},
		Retry:           s.Retry,
		Log:             s.Log,
	}

	var err error
	newStorage.conn, err = redis.Dial("tcp", s.Addr.Host+":"+s.Addr.Port)
	for err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "Init",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.Retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		newStorage.conn, err = redis.Dial("tcp", s.Addr.Host+":"+s.Addr.Port)
	}

	return &newStorage, err
}

func (s *RedisStorage) CloseConn() error {
	return s.conn.Close()
}

func (s *RedisStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := redis.Strings(s.conn.Do("SMEMBERS", "devParamsKeys"))
	if err != nil {
		errors.Wrap(err, "RedisStorage: GetDevsData(): SMembers for devParamsKeys has failed")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for k, v := range devParamsKeys {
		devParamsKeysTokens[k] = strings.Split(v, ":")
	}

	var (
		devData  entities.DevData
		devsData []entities.DevData
	)

	for index, key := range devParamsKeysTokens {
		devData.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		devData.Data = make(map[string][]string)

		params, err := redis.Strings(s.conn.Do("SMEMBERS", devParamsKeys[index]))
		if err != nil {
			errors.Wrapf(err, "RedisStorage: GetDevsData(): SMembers() for %s has failed", devParamsKeys[index])
		}

		vals := make([][]string, len(params))
		for i, p := range params {
			vals[i], _ = redis.Strings(s.conn.Do("ZRANGEBYSCORE", devParamsKeys[index]+":"+p, "-inf", "inf"))
			if err != nil {
				errors.Wrapf(err, "RedisStorage: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
			}
			devData.Data[p] = vals[i]
		}
		devsData = append(devsData, devData)
	}
	return devsData, err
}

func (s *RedisStorage) GetDevMeta(id entities.DevID) (*entities.DevMeta, error) {
	t, err := redis.String(s.conn.Do("HGET", "id:"+id, "type"))
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta(): HGet() has failed")
	}
	n, err := redis.String(s.conn.Do("HGET", "id:"+id, "name"))
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta():  HGet() has failed")
	}
	m, err := redis.String(s.conn.Do("HGET", "id:"+id, "mac"))
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta():  HGet() has failed")
	}

	meta := entities.DevMeta{
		Type: t,
		Name: n,
		MAC:  m,
	}
	return &meta, nil
}

func (s *RedisStorage) SetDevMeta(m *entities.DevMeta) error {
	if _, err := s.conn.Do("HMSET", "id:"+m.MAC, "type", m.Type); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := s.conn.Do("HMSET", "id:"+m.MAC, "name", m.Name); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := s.conn.Do("HMSET", "id:"+m.MAC, "mac", m.MAC); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	return nil
}

func (s *RedisStorage) DevIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + partialDevConfigKey
	if ok, err := redis.Bool(s.conn.Do("EXISTS", configKey)); ok {
		if err != nil {
			errors.Wrap(err, "RedisStorage: DevIsRegistered(): Exists() has failed")
		}
		return true, err
	}
	return false, nil
}

func (s *RedisStorage) GetDevData(id entities.DevID) (*entities.DevData, error) {
	meta, err := s.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch meta.Type {
	case "fridge":
		return s.getFridgeData(meta)
	case "washer":
		return s.getWasherData(meta)
	default:
		return &entities.DevData{}, errors.New("RedisStorage: GetDevData(): dev type is unknown")
	}
}

func (s *RedisStorage) SaveDevData(r *entities.RawDevData) error {
	switch r.Meta.Type {
	case "fridge":
		return s.saveFridgeData(r)
	case "washer":
		return s.saveWasherData(r)
	default:
		return errors.New("RedisStorage: SaveDevData(): dev type is unknown")
	}
}

func (s *RedisStorage) GetDevConfig(id entities.DevID) (*entities.DevConfig, error) {
	meta, err := s.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch meta.Type {
	case "fridge":
		return s.getFridgeConfig(meta)
	case "washer":
		return s.getWasherConfig(meta)
	default:
		return &entities.DevConfig{}, errors.New("RedisStorage: GetDevConfig(): dev type is unknown")
	}
}

func (s *RedisStorage) SetDevConfig(id entities.DevID, c *entities.DevConfig) error {
	meta, err := s.GetDevMeta(id)
	if err != nil {
		return err
	}

	switch meta.Type {
	case "fridge":
		return s.setFridgeConfig(c, meta)
	case "washer":
		return s.setWasherConfig(c)
	default:
		return errors.New("RedisStorage: SetDevConfig(): dev type is unknown")
	}
}

func (s *RedisStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return s.getFridgeDefaultConfig(m)
	case "washer":
		return s.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("RedisStorage: GetDevDefaultConfig(): dev type is unknown")
	}
}

// Publish posts a message on the given channel.
func (s *RedisStorage) Publish(msg interface{}, channel string) (int64, error) {
	numberOfClients, err := redis.Int64(s.conn.Do("PUBLISH", msg, channel))
	if err != nil {
		return 0, err
	}
	return numberOfClients, nil
}

// Subscribe subscribes the client to the specified channels.
func (s *RedisStorage) Subscribe(cn chan []byte, channel ...string) {
	psc := redis.PubSubConn{Conn: s.conn}
	for _, c := range channel {
		psc.Subscribe(c)
	}

	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			s.Log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Infof("chan %s: message: %s", v.Channel, v.Data)
			cn <- v.Data
		case redis.Subscription:
			s.Log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Infof("chan %s: kind: %s count: %d", v.Channel, v.Kind, v.Count)
		case error:
			s.Log.WithFields(logrus.Fields{
				"func": "Subscribe",
			}).Errorf("%s", v)
		}
	}
}
