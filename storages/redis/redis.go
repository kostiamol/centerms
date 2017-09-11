package storages

import (
	"encoding/json"
	"strings"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"
)

type RedisDevStore struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rs *RedisDevStore) SetServer(s *entities.Server) error {
	var err error
	if s.Host == "" {
		err = errors.New("RedisDevStore: SetServer(): store server's host is empty")
	} else if s.Port == 0 {
		err = errors.Wrap(err, "RedisDevStore: SetServer(): store server's port is empty")
	}

	rs.DbServer = *s
	return err
}

func (rs *RedisDevStore) CreateConn() (entities.DevStore, error) {
	nrc := RedisDevStore{
		Client:   redis.New(),
		DbServer: rs.DbServer,
	}

	err := nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	for err != nil {
		log.Errorln("RedisDevStore: CreateConn(): Connect() has failed")
		time.Sleep(3 * time.Second)
		err = nrc.Client.Connect(nrc.DbServer.Host, nrc.DbServer.Port)
	}

	return &nrc, err
}

func (rs *RedisDevStore) CloseConn() error {
	return rs.Client.Close()
}

func (rs *RedisDevStore) GetDevsData() ([]entities.DevData, error) {
	devParamsKeys, err := rs.Client.SMembers("devParamsKeys")
	if err != nil {
		errors.Wrap(err, "RedisDevStore: GetDevsData(): SMembers for devParamsKeys has failed")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for i, k := range devParamsKeys {
		devParamsKeysTokens[i] = strings.Split(k, ":")
	}

	var (
		d  entities.DevData
		ds []entities.DevData
	)

	for i, k := range devParamsKeysTokens {
		params, err := rs.Client.SMembers(devParamsKeys[i])
		if err != nil {
			errors.Wrapf(err, "RedisDevStore: GetDevsData(): SMembers() for %s has failed", devParamsKeys[i])
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
				errors.Wrapf(err, "RedisDevStore: GetDevsData(): ZRangeByScore() for %s has failed", devParamsKeys[i])
			}
			d.Data[p] = values[i]
		}
		ds = append(ds, d)
	}
	return ds, err
}

func (rs *RedisDevStore) GetDevData(m *entities.DevMeta) (*entities.DevData, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeData(m)
	case "washer":
		return rs.getWasherData(m)
	default:
		return &entities.DevData{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStore) SetDevData(r *entities.Request) error {
	switch r.Meta.Type {
	case "fridge":
		return rs.setFridgeData(r)
	case "washer":
		return rs.setWasherData(r)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStore) GetDevConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeConfig(m)
	case "washer":
		return rs.getWasherConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStore) SetDevConfig(m *entities.DevMeta, c *entities.DevConfig) error {
	switch m.Type {
	case "fridge":
		return rs.setFridgeConfig(m.MAC, c)
	case "washer":
		return rs.setWasherConfig(m.MAC, c)
	default:
		return errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStore) GetDevDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	switch m.Type {
	case "fridge":
		return rs.getFridgeDefaultConfig(m)
	case "washer":
		return rs.getWasherDefaultConfig(m)
	default:
		return &entities.DevConfig{}, errors.New("dev type is unknown")
	}
}

func (rs *RedisDevStore) Publish(channel string, msg interface{}) (int64, error) {
	return rs.Client.Publish(channel, msg)
}

func (rs *RedisDevStore) Subscribe(c chan []string, channel ...string) error {
	return rs.Client.Subscribe(c, channel...)
}

func PublishWS(r *entities.Request, roomID string, st entities.DevStore) error {
	pr, err := json.Marshal(r)
	for err != nil {
		errors.Wrap(err, "RedisDevStore: PublishWS(): Request marshalling has failed")
	}

	conn, err := st.CreateConn()
	if err != nil {
		errors.Wrap(err, "RedisDevStore: PublishWS(): storage connection hasn't been established")
	}
	defer conn.CloseConn()

	_, err = st.Publish(roomID, pr)
	if err != nil {
		errors.Wrap(err, "RedisDevStore: PublishWS(): publishing has failed")
	}

	return err
}

func (rs *RedisDevStore) devIsRegistered(m *entities.DevMeta) (bool, error) {
	configKey := m.MAC + ":" + "config"
	if ok, err := rs.Client.Exists(configKey); ok {
		return true, err
	}
	return false, nil
}