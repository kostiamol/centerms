package storages

import (
	"encoding/json"
	"strings"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

const (
	partKeyToConfig = ":config"
)

type RedisStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rc *RedisStorage) SetServer(s entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("db server host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "db server port is empty")
	}

	rc.DbServer = s
	return err
}

func (rc *RedisStorage) FlushAll() error {
	_, err := rc.Client.FlushAll()
	return err
}

func (rc *RedisStorage) Publish(channel string, message interface{}) (int64, error) {
	return rc.Client.Publish(channel, message)
}

func (rc *RedisStorage) Subscribe(c chan []string, channel ...string) error {
	return rc.Client.Subscribe(c, channel...)
}

func (rc *RedisStorage) CloseConnection() error {
	return rc.Client.Close()
}

func (rc *RedisStorage) CreateConnection() (entities.Storage, error) {
	nrc := RedisStorage{
		Client:   redis.New(),
		DbServer: rc.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	return &nrc, err
}

func PublishWS(r *entities.Request, roomID string, db entities.Storage) error {
	pubReq, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "req marshalling has failed")
	}

	conn, err := db.CreateConnection()
	if err != nil {
		errors.Wrap(err, "db connection hasn't been established")
	}
	defer conn.CloseConnection()

	_, err = db.Publish(roomID, pubReq)
	if err != nil {
		errors.Wrap(err, "publishing has failed")
	}

	return err
}

func (rc *RedisStorage) GetAllDevices() ([]entities.DevData, error) {
	rc.CreateConnection()

	var device entities.DevData
	var devices []entities.DevData

	devParamsKeys, err := rc.Client.SMembers("devParamsKeys")
	if err != nil {
		errors.Wrap(err, "can't read set members from devParamsKeys")
	}

	var devParamsKeysTokens = make([][]string, len(devParamsKeys))
	for i, k := range devParamsKeys {
		devParamsKeysTokens[i] = strings.Split(k, ":")
	}

	for index, key := range devParamsKeysTokens {
		params, err := rc.Client.SMembers(devParamsKeys[index])
		if err != nil {
			errors.Wrapf(err, "can't read members from %s", devParamsKeys[index])
		}

		device.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		device.Data = make(map[string][]string)

		values := make([][]string, len(params))
		for i, p := range params {
			values[i], _ = rc.Client.ZRangeByScore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "can't use ZRangeByScore for %s", devParamsKeys[index])
			}
			device.Data[p] = values[i]
		}

		devices = append(devices, device)
	}
	return devices, err
}

func (rc *RedisStorage) GetKeyForConfig(mac string) (string, error) {
	return mac + partKeyToConfig, nil
}

func (rc *RedisStorage) GetDevData(devParamsKey string, m entities.DevMeta) (entities.DevData, error) {
	switch t := m.Type; t {
	case "fridge":
		return rc.getFridgeData(devParamsKey, m)
	default:
		return entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevData(m entities.DevMeta, r *entities.Request) error {
	switch t := m.Type; t {
	case "fridge":
		return rc.setFridgeData(r)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevConfig(t string, configInfo, mac string) (*entities.DevConfig, error) {
	switch t := t; t {
	case "fridge":
		return rc.getFridgeConfig(configInfo, mac)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevConfig(t string, configInfo string, c *entities.DevConfig) error {
	switch t := t; t {
	case "fridge":
		return rc.setFridgeConfig(configInfo, c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevDefaultConfig(t string, f *entities.Fridge) (*entities.DevConfig, error) {
	switch t := t; t {
	case "fridge":
		return rc.getFridgeDefaultConfig(f)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}
