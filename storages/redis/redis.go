package storages

import (
	"strings"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

const (
	partialDevKey       = "device:"
	partialDevConfigKey = ":config"
	partialDevParamsKey = ":params"
)

type RedisStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rds *RedisStorage) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisStorage: SetServer(): store server'rds host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "RedisStorage: SetServer(): store server'rds port is empty")
	}
	rds.DbServer = *s

	return err
}

func (rds *RedisStorage) CreateConn() (entities.DevStorage, error) {
	nrc := RedisStorage{
		Client:   redis.New(),
		DbServer: rds.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	for err != nil {
		log.Errorln("RedisStorage: CreateConn(): Connect() has failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
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
		dd  entities.DevData
		dsd []entities.DevData
	)

	for index, key := range devParamsKeysTokens {
		params, err := rds.Client.SMembers(devParamsKeys[index])
		if err != nil {
			errors.Wrapf(err, "RedisStorage: GetDevsData(): SMembers() for %rds has failed", devParamsKeys[index])
		}

		dd.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		dd.Data = make(map[string][]string)

		vals := make([][]string, len(params))
		for i, p := range params {
			vals[i], _ = rds.Client.ZRangeByScore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "RedisStorage: GetDevsData(): ZRangeByScore() for %rds has failed", devParamsKeys[i])
			}
			dd.Data[p] = vals[i]
		}
		dsd = append(dsd, dd)
	}

	return dsd, err
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

func (rds *RedisStorage) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeData(m)
	case "washer":
		return rds.getWasherData(m)
	default:
		return &entities.DevData{}, errors.New("RedisStorage: GetDevData(): dev type is unknown")
	}
}

func (rds *RedisStorage) SaveDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rds.saveFridgeData(r)
	case "washer":
		return rds.saveWasherData(r)
	default:
		return errors.New("RedisStorage: SaveDevData(): dev type is unknown")
	}
}

func (rds *RedisStorage) GetDevConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeConfig(m)
	case "washer":
		return rds.getWasherConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("RedisStorage: GetDevConfig(): dev type is unknown")
	}
}

func (rds *RedisStorage) SetDevConfig(m *entities.DevMeta, c *entities.DevConfig) error {
	switch m.Type {
	case "fridge":
		return rds.setFridgeConfig(c, m)
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

func (rds *RedisStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rds.Client.Publish(channel, msg)
}

func (rds *RedisStorage) Subscribe(c chan []string, channel ...string) error {
	return rds.Client.Subscribe(c, channel...)
}
