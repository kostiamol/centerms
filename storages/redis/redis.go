package storages

import (
	"strings"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

type RedisDevStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rs *RedisDevStorage) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisDevStorage: SetServer(): store server's host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "RedisDevStorage: SetServer(): store server's port is empty")
	}
	rs.DbServer = *s

	return err
}

func (rs *RedisDevStorage) CreateConn() (entities.DevStorage, error) {
	nrc := RedisDevStorage{
		Client:   redis.New(),
		DbServer: rs.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	for err != nil {
		log.Errorln("RedisDevStorage: CreateConn(): Connect() has failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	}

	return &nrc, err
}

func (rs *RedisDevStorage) CloseConn() error {
	return rs.Client.Close()
}

func (rs *RedisDevStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := rs.Client.SMembers("devParamsKeys")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: GetDevsData(): SMembers for devParamsKeys has failed")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for k, v := range devParamsKeys {
		devParamsKeysTokens[k] = strings.Split(v, ":")
	}

	var (
		d  entities.DevData
		ds []entities.DevData
	)

	for i, k := range devParamsKeysTokens {
		params, err := rs.Client.SMembers(devParamsKeys[i])
		if err != nil {
			errors.Wrapf(err, "RedisDevStorage: GetDevsData(): SMembers() for %s has failed", devParamsKeys[i])
		}

		d.Meta = entities.DevMeta{
			Type: k[1],
			Name: k[2],
			MAC:  k[3],
		}
		d.Data = make(map[string][]string)

		vals := make([][]string, len(params))
		for i, p := range params {
			vals[i], _ = rs.Client.ZRangeByScore(devParamsKeys[i]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "RedisDevStorage: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
			}
			d.Data[p] = vals[i]
		}
		ds = append(ds, d)
	}

	return ds, err
}

func (rs *RedisDevStorage) DevIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + ":" + "config"
	if ok, err := rs.Client.Exists(configKey); ok {
		return true, err
	}

	return false, nil
}

func (rs *RedisDevStorage) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeData(m)
	case "washer":
		return rs.getWasherData(m)
	default:
		return &entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStorage) SetDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rs.setFridgeData(r)
	case "washer":
		return rs.setWasherData(r)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStorage) GetDevConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeConfig(m)
	case "washer":
		return rs.getWasherConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStorage) SetDevConfig(m *entities.DevMeta, c *entities.DevConfig) error {
	switch m.Type {
	case "fridge":
		return rs.setFridgeConfig(c)
	case "washer":
		return rs.setWasherConfig(c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeDefaultConfig(m)
	case "washer":
		return rs.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rs.Client.Publish(channel, msg)
}

func (rs *RedisDevStorage) Subscribe(c chan []string, channel ...string) error {
	return rs.Client.Subscribe(c, channel...)
}
