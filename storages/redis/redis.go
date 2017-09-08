package storages

import (
	"encoding/json"
	"strings"

	"net"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

type RedisStorage struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rs *RedisStorage) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisStorage: SetServer(): storage server's host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "RedisStorage: SetServer(): storage server's port is empty")
	}

	rs.DbServer = *s
	return err
}

func (rs *RedisStorage) Publish(channel string, msg interface{}) (int64, error) {
	return rs.Client.Publish(channel, msg)
}

func (rs *RedisStorage) Subscribe(c chan []string, channel ...string) error {
	return rs.Client.Subscribe(c, channel...)
}

func (rs *RedisStorage) CloseConn() error {
	return rs.Client.Close()
}

func (rs *RedisStorage) CreateConn() (entities.Storage, error) {
	nrc := RedisStorage{
		Client:   redis.New(),
		DbServer: rs.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	for err != nil {
		log.Errorln("RedisStorage: CreateConn(): Connect() has failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	}

	return &nrc, err
}

func PublishWS(r *entities.Request, roomID string, st entities.Storage) error {
	pr, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "RedisStorage: PublishWS(): Request marshalling has failed")
	}

	conn, err := st.CreateConn()
	if err != nil {
		errors.Wrap(err, "RedisStorage: PublishWS(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	_, err = st.Publish(roomID, pr)
	if err != nil {
		errors.Wrap(err, "RedisStorage: PublishWS(): publishing has failed")
	}

	return err
}

func (rs *RedisStorage) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := rs.Client.SMembers("devParamsKeys")
	if err != nil {
		errors.Wrap(err, "RedisStorage: GetDevsData(): SMembers for devParamsKeys has failed")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for i, k := range devParamsKeys {
		devParamsKeysTokens[i] = strings.Split(k, ":")
	}

	var (
		d entities.DevData
		ds []entities.DevData
	)

	for i, k := range devParamsKeysTokens {
		params, err := rs.Client.SMembers(devParamsKeys[i])
		if err != nil {
			errors.Wrapf(err, "RedisStorage: GetDevsData(): SMembers() for %s has failed", devParamsKeys[i])
		}

		d.Meta = entities.DevMeta{
			Type: k[1],
			Name: k[2],
			MAC:  k[3],
		}
		d.Data = make(map[string][]string)

		values := make([][]string, len(params))
		for i, p := range params {
			values[i], _ = rs.Client.ZRangeByScore(devParamsKeys[i]+":"+p, "-inf", "inf")
			if err != nil {
				errors.Wrapf(err, "RedisStorage: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
			}
			d.Data[p] = values[i]
		}
		ds = append(ds, d)
	}
	return ds, err
}

func (rs *RedisStorage) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeData(m)
	case "washer":
		return rs.getWasherData(m)
	default:
		return &entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisStorage) SetDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rs.setFridgeData(r)
	case "washer":
		return rs.setWasherData(r)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisStorage) GetDevConfig(t string, mac string) (*entities.DevConfig, error) {
	switch t {
	case "fridge":
		return rs.getFridgeConfig(mac)
	case "washer":
		return rs.getWasherConfig(mac)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisStorage) SetDevConfig(t string, mac string, c *entities.DevConfig) error {
	switch t {
	case "fridge":
		return rs.setFridgeConfig(mac, c)
	case "washer":
		return rs.setWasherConfig(mac, c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisStorage) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeDefaultConfig(m)
	case "washer":
		return rs.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisStorage) SendDevDefaultConfig(c *net.Conn, r *entities.Request) ([]byte, error) {
	switch r.Meta.Type {
	case "fridge":
		return rs.sendFridgeDefaultConfig(c, r)
	case "washer":
		return rs.sendWasherDefaultConfig(c, r)
	default:
		return []byte{}, errors.New("dev type is unknown")
	}
}
