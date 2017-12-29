package storages

import (
	"math/rand"
	"strings"

	"time"

	"strconv"

	"github.com/Sirupsen/logrus"
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
	Client        *redis.Client
	DbServer      entities.Server
	RetryInterval time.Duration
}

func (rds *RedisStorage) SetServer(s entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisStorage: SetServer(): store server'rds host is empty")
	} else if s.Port == "" {
		err = errors.Wrap(err, "RedisStorage: SetServer(): store server'rds port is empty")
	}
	rds.DbServer = s
	return err
}

func (rds *RedisStorage) CreateConn() (entities.Storage, error) {
	nrc := RedisStorage{
		Client:   redis.New(),
		DbServer: rds.DbServer,
	}

	parsedPort, err := strconv.ParseUint(nrc.DbServer.Port, 10, 64)
	if err != nil {
		errors.New("RedisStorage: CreateConn(): ParseUint() has failed")
	}
	port := uint(parsedPort)

	err = nrc.Client.Connect(nrc.DbServer.Host, port)
	for err != nil {
		logrus.Errorf("RedisStorage: CreateConn(): Connect() has failed: %s", err)
		duration := time.Duration(rand.Intn(int(rds.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		err = nrc.Client.Connect(nrc.DbServer.Host, port)
	}
	return &nrc, err
}

func (rds *RedisStorage) CloseConn() error {
	return rds.Client.Close()
}

func (rds *RedisStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := rds.Client.SMembers("devParamsKeys")
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
		params, err := rds.Client.SMembers(devParamsKeys[index])
		if err != nil {
			errors.Wrapf(err, "RedisStorage: GetDevsData(): SMembers() for %rds has failed", devParamsKeys[index])
		}

		devData.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		devData.Data = make(map[string][]string)

		vals := make([][]string, len(params))
		for i, p := range params {
			vals[i], _ = rds.Client.ZRangeByScore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "RedisStorage: GetDevsData(): ZRangeByScore() for %rds has failed", devParamsKeys[i])
			}
			devData.Data[p] = vals[i]
		}
		devsData = append(devsData, devData)
	}
	return devsData, err
}

func (rds *RedisStorage) GetDevMeta(id string) (*entities.DevMeta, error) {
	t, err := rds.Client.HGet("id:"+id, "type")
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta(): HGet() has failed")
	}
	n, err := rds.Client.HGet("id:"+id, "name")
	if err != nil {
		errors.Wrapf(err, "RedisStorage: GetDevMeta():  HGet() has failed")
	}
	m, err := rds.Client.HGet("id:"+id, "mac")
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

func (rds *RedisStorage) SetDevMeta(m *entities.DevMeta) error {
	if _, err := rds.Client.HMSet("id:"+m.MAC, "type", m.Type); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := rds.Client.HMSet("id:"+m.MAC, "name", m.Name); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	if _, err := rds.Client.HMSet("id:"+m.MAC, "mac", m.MAC); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		return err
	}
	return nil
}

func (rds *RedisStorage) DevIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + partialDevConfigKey
	if ok, err := rds.Client.Exists(configKey); ok {
		if err != nil {
			errors.Wrap(err, "RedisStorage: DevIsRegistered(): Exists() has failed")
		}
		return true, err
	}
	return false, nil
}

func (rds *RedisStorage) GetDevData(id string) (*entities.DevData, error) {
	meta, err := rds.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch meta.Type {
	case "fridge":
		return rds.getFridgeData(meta)
	case "washer":
		return rds.getWasherData(meta)
	default:
		return &entities.DevData{}, errors.New("RedisStorage: GetDevData(): dev type is unknown")
	}
}

func (rds *RedisStorage) SaveDevData(r *entities.SaveDevDataRequest) error {
	switch r.Meta.Type {
	case "fridge":
		return rds.saveFridgeData(r)
	case "washer":
		return rds.saveWasherData(r)
	default:
		return errors.New("RedisStorage: SaveDevData(): dev type is unknown")
	}
}

func (rds *RedisStorage) GetDevConfig(id string) (*entities.DevConfig, error) {
	meta, err := rds.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch meta.Type {
	case "fridge":
		return rds.getFridgeConfig(meta)
	case "washer":
		return rds.getWasherConfig(meta)
	default:
		return &entities.DevConfig{}, errors.New("RedisStorage: GetDevConfig(): dev type is unknown")
	}
}

func (rds *RedisStorage) SetDevConfig(id string, c *entities.DevConfig) error {
	meta, err := rds.GetDevMeta(id)
	if err != nil {
		return err
	}

	switch meta.Type {
	case "fridge":
		return rds.setFridgeConfig(c, meta)
	case "washer":
		return rds.setWasherConfig(c)
	default:
		return errors.New("RedisStorage: SetDevConfig(): dev type is unknown")
	}
}

func (rds *RedisStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeDefaultConfig(m)
	case "washer":
		return rds.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("RedisStorage: GetDevDefaultConfig(): dev type is unknown")
	}
}

// Publish posts a message on the given subject.
func (rds *RedisStorage) Publish(subject string, message interface{}) (int64, error) {
	return rds.Client.Publish(subject, message)
}

// Subscribe subscribes the client to the specified subjects.
func (rds *RedisStorage) Subscribe(cn chan []string, subject ...string) error {
	return rds.Client.Subscribe(cn, subject...)
}
