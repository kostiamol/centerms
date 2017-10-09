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

type RedisDevStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rds *RedisDevStorage) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisDevStorage: SetServer(): store server'rds host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "RedisDevStorage: SetServer(): store server'rds port is empty")
	}
	rds.DbServer = *s

	return err
}

func (rds *RedisDevStorage) CreateConn() (entities.DevStorage, error) {
	nrc := RedisDevStorage{
		Client:   redis.New(),
		DbServer: rds.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	for err != nil {
		log.Errorln("RedisDevStorage: CreateConn(): Connect() has failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	}

	return &nrc, err
}

func (rds *RedisDevStorage) CloseConn() error {
	return rds.Client.Close()
}

func (rds *RedisDevStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := rds.Client.SMembers("devParamsKeys")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: GetDevsData(): SMembers for devParamsKeys has failed")
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
			errors.Wrapf(err, "RedisDevStorage: GetDevsData(): SMembers() for %rds has failed", devParamsKeys[index])
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
				errors.Wrapf(err, "RedisDevStorage: GetDevsData(): ZRangeByScore() for %rds has failed", devParamsKeys[i])
			}
			dd.Data[p] = vals[i]
		}
		dsd = append(dsd, dd)
	}

	return dsd, err
}

func (rds *RedisDevStorage) DevIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + partialDevConfigKey
	if ok, err := rds.Client.Exists(configKey); ok {
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: DevIsRegistered(): Exists() has failed")
		}
		return true, err
	}

	return false, nil
}

func (rds *RedisDevStorage) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeData(m)
	case "washer":
		return rds.getWasherData(m)
	default:
		return &entities.DevData{}, errors.New("RedisDevStorage: GetDevData(): dev type is unknown")
	}
}

func (rds *RedisDevStorage) SaveDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rds.saveFridgeData(r)
	case "washer":
		return rds.saveWasherData(r)
	default:
		return errors.New("RedisDevStorage: SaveDevData(): dev type is unknown")
	}
}

func (rds *RedisDevStorage) GetDevConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeConfig(m)
	case "washer":
		return rds.getWasherConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("RedisDevStorage: GetDevConfig(): dev type is unknown")
	}
}

func (rds *RedisDevStorage) SetDevConfig(m *entities.DevMeta, c *entities.DevConfig) error {
	switch m.Type {
	case "fridge":
		return rds.setFridgeConfig(c, m)
	case "washer":
		return rds.setWasherConfig(c)
	default:
		return errors.New("RedisDevStorage: SetDevConfig(): dev type is unknown")
	}
}

func (rds *RedisDevStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rds.getFridgeDefaultConfig(m)
	case "washer":
		return rds.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("RedisDevStorage: GetDevDefaultConfig(): dev type is unknown")
	}
}

func (rds *RedisDevStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rds.Client.Publish(channel, msg)
}

func (rds *RedisDevStorage) Subscribe(c chan []string, channel ...string) error {
	return rds.Client.Subscribe(c, channel...)
}
