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
	partialConfigKey = ":config"
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

func (rc *RedisStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rc.Client.Publish(channel, msg)
}

func (rc *RedisStorage) Subscribe(c chan []string, channel ...string) error {
	return rc.Client.Subscribe(c, channel...)
}

func (rc *RedisStorage) CloseConn() error {
	return rc.Client.Close()
}

func (rc *RedisStorage) CreateConn() (entities.Storage, error) {
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

func PublishWS(r *entities.Request, roomID string, st entities.Storage) error {
	pubReq, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "req marshalling has failed")
	}

	conn, err := st.CreateConn()
	if err != nil {
		errors.Wrap(err, "PublishWS(): db connection hasn't been established")
	}
	defer conn.CloseConn()

	_, err = st.Publish(roomID, pubReq)
	if err != nil {
		errors.Wrap(err, "publishing has failed")
	}

	return err
}

func (rc *RedisStorage) GetDevsData() ([]entities.DevData, error) {
	rc.CreateConn()

	var d entities.DevData
	var ds []entities.DevData

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

		d.Meta = entities.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		d.Data = make(map[string][]string)

		values := make([][]string, len(params))
		for i, p := range params {
			values[i], _ = rc.Client.ZRangeByScore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "can't use ZRangeByScore for %s", devParamsKeys[index])
			}
			d.Data[p] = values[i]
		}
		ds = append(ds, d)
	}
	return ds, err
}

func (rc *RedisStorage) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	devID := "device:" + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devID + ":" + "params"

	switch m.Type {
	case "fridge":
		return rc.getFridgeData(devParamsKey, m)
	case "washer":
		return rc.getWasherData(devParamsKey, m)
	default:
		return &entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rc.setFridgeData(r)
	case "washer":
		return rc.setWasherData(r)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevConfig(t string, mac string) (*entities.DevConfig, error) {
	k := mac + partialConfigKey
	switch t {
	case "fridge":
		return rc.getFridgeConfig(k, mac)
	case "washer":
		return rc.getWasherConfig(k, mac)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SetDevConfig(t string, mac string, c *entities.DevConfig) error {
	k := mac + partialConfigKey
	switch t {
	case "fridge":
		return rc.setFridgeConfig(k, c)
	case "washer":
		return rc.setWasherConfig(k, c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rc.getFridgeDefaultConfig(m)
	case "washer":
		return rc.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rc *RedisStorage) SendDevDefaultConfig(c *net.Conn, r *entities.Request) ([]byte, error) {
	switch r.Meta.Type {
	case "fridge":
		return rc.sendFridgeDefaultConfig(c, r)
	case "washer":
		return rc.sendWasherDefaultConfig(c, r)
	default:
		return []byte{}, errors.New("dev type is unknown")
	}
}
