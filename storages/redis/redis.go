package storages

import (
	"math/rand"
	"strings"

	"time"

	"strconv"

	"github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/kostiamol/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

const (
	partialDevKey       = "device:"
	partialDevConfigKey = ":config"
	partialDevParamsKey = ":params"
)

type RedisStorage struct {
	entities.ExternalService
	Client *redis.Client
	Retry  time.Duration
	Log    *logrus.Entry
}

func NewRedisStorage(addr entities.Address, name string, ttl time.Duration, retry time.Duration,
	log *logrus.Entry) entities.Storager {

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

	parsedPort, err := strconv.ParseUint(s.Addr.Port, 10, 64)
	if err != nil {
		return errors.New("RedisStorage: CreateConn(): ParseUint() has failed")
	}
	port := uint(parsedPort)

	s.Client = redis.New()
	err = s.Client.Connect(s.Addr.Host, port)
	for err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "Init",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.Retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		err = s.Client.Connect(s.Addr.Host, port)
	}

	if ok, err := s.Check(); !ok {
		return err
	}

	c, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return err
	}
	s.ConsulAgent = c.Agent()

	serviceDef := &consul.AgentServiceRegistration{
		Name: s.Name,
		Check: &consul.AgentServiceCheck{
			TTL: s.TTL.String(),
		},
	}

	if err := s.ConsulAgent.ServiceRegister(serviceDef); err != nil {
		return err
	}
	go s.UpdateTTL(s.Check)

	return nil
}

// Check method issues PING Redis command to check if we’re ok.
func (s *RedisStorage) Check() (bool, error) {
	_, err := s.Client.Ping()
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
	parsedPort, err := strconv.ParseUint(s.Addr.Port, 10, 64)
	if err != nil {
		errors.New("RedisStorage: CreateConn(): ParseUint() has failed")
	}
	port := uint(parsedPort)

	newStorage := RedisStorage{
		ExternalService: entities.ExternalService{Addr: s.Addr},
		Client:          redis.New(),
		Retry:           s.Retry,
	}
	err = newStorage.Client.Connect(newStorage.Addr.Host, port)
	for err != nil {
		s.Log.WithFields(logrus.Fields{
			"func": "Init",
		}).Errorf("Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(s.Retry.Seconds())))
		time.Sleep(time.Second*duration + 1)
		err = newStorage.Client.Connect(newStorage.Addr.Host, port)
	}

	return &newStorage, err
}

func (s *RedisStorage) CloseConn() error {
	return s.Client.Close()
}

func (s *RedisStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := s.Client.SMembers("devParamsKeys")
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
		params, err := s.Client.SMembers(devParamsKeys[index])
		if err != nil {
			errors.Wrapf(err, "RedisStorage: GetDevsData(): SMembers() for %s has failed", devParamsKeys[index])
		}

		devData.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		devData.Data = make(map[string][]string)

		vals := make([][]string, len(params))
		for i, p := range params {
			vals[i], _ = s.Client.ZRangeByScore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "RedisStorage: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
			}
			devData.Data[p] = vals[i]
		}
		devsData = append(devsData, devData)
	}
	return devsData, err
}

func (s *RedisStorage) GetDevMeta(id string) (*entities.DevMeta, error) {
	t, err := s.Client.HGet("id:"+id, "type")
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta(): HGet() has failed")
	}
	n, err := s.Client.HGet("id:"+id, "name")
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta():  HGet() has failed")
	}
	m, err := s.Client.HGet("id:"+id, "mac")
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
	if _, err := s.Client.HMSet("id:"+m.MAC, "type", m.Type); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := s.Client.HMSet("id:"+m.MAC, "name", m.Name); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := s.Client.HMSet("id:"+m.MAC, "mac", m.MAC); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	return nil
}

func (s *RedisStorage) DevIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + partialDevConfigKey
	if ok, err := s.Client.Exists(configKey); ok {
		if err != nil {
			errors.Wrap(err, "RedisStorage: DevIsRegistered(): Exists() has failed")
		}
		return true, err
	}
	return false, nil
}

func (s *RedisStorage) GetDevData(id string) (*entities.DevData, error) {
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

func (s *RedisStorage) SaveDevData(r *entities.SaveDevDataRequest) error {
	switch r.Meta.Type {
	case "fridge":
		return s.saveFridgeData(r)
	case "washer":
		return s.saveWasherData(r)
	default:
		return errors.New("RedisStorage: SaveDevData(): dev type is unknown")
	}
}

func (s *RedisStorage) GetDevConfig(id string) (*entities.DevConfig, error) {
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

func (s *RedisStorage) SetDevConfig(id string, c *entities.DevConfig) error {
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

// Publish posts a message on the given subject.
func (s *RedisStorage) Publish(subject string, message interface{}) (int64, error) {
	return s.Client.Publish(subject, message)
}

// Subscribe subscribes the client to the specified subjects.
func (s *RedisStorage) Subscribe(cn chan []string, subject ...string) error {
	return s.Client.Subscribe(cn, subject...)
}
