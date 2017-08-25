package storages

import (
	"encoding/json"
	"strings"

	"net"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
	"time"
	log "github.com/Sirupsen/logrus"
)

const (
	partKeyToConfig = ":config"
)

type RedisStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rc *RedisStorage) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("db server host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "db server port is empty")
	}

	rc.DbServer = *s
	return err
}

func (rc *RedisStorage) FlushAll() error {
	_, err := rc.Client.FlushAll()
	return err
}

func (rc *RedisStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rc.Client.Publish(channel, msg)
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
	for err != nil {
		log.Errorln("RedisStorage: Connect(): failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	}

	return &nrc, err
}

func PublishWS(r *entities.Request, roomID string, db entities.Storage) error {
	pubReq, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "req marshalling has failed")
	}

	conn, err := db.CreateConnection()
	if err != nil {
		errors.Wrap(err, "PublishWS(): db connection hasn't been established")
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

func (rc *RedisStorage) GetDevData(devParamsKey string, m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rc.getFridgeData(devParamsKey, m)
	case "washer":
		return rc.getWasherData(devParamsKey, m)
	default:
		return &entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevData(req *entities.Request) error {
	switch req.Meta.Type {
	case "fridge":
		return rc.setFridgeData(req)
	case "washer":
		return rc.setWasherData(req)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevConfig(t string, configInfo string, mac string) (*entities.DevConfig, error) {
	switch t {
	case "fridge":
		return rc.getFridgeConfig(configInfo, mac)
	case "washer":
		return rc.getWasherConfig(configInfo, mac)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevConfig(t string, configInfo string, c *entities.DevConfig) error {
	switch t {
	case "fridge":
		return rc.setFridgeConfig(configInfo, c)
	case "washer":
		return rc.setWasherConfig(configInfo, c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevDefaultConfig(t string, m *entities.DevMeta) (*entities.DevConfig, error) {
	switch t {
	case "fridge":
		return rc.getFridgeDefaultConfig(m)
	case "washer":
		return rc.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SendDevDefaultConfig(c *net.Conn, req *entities.Request) ([]byte, error) {
	switch req.Meta.Type {
	case "fridge":
		return rc.sendFridgeDefaultConfig(c, req)
	case "washer":
		return rc.sendWasherDefaultConfig(c, req)
	default:
		return []byte{}, errors.New("dev type is unknown")
	}
}
