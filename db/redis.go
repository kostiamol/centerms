package db

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"menteslibres.net/gosexy/redis"

	"github.com/giperboloid/centerms/entities"
)

const (
	partKeyToConfig = ":config"
)

type RedisClient struct {
	Client   *redis.Client
	DbServer entities.Server
}

func (rc *RedisClient) SetDBServer(server entities.Server) {
	rc.DbServer = server
}

func (rc *RedisClient) FlushAll() error {
	_, err := rc.Client.FlushAll()
	return err
}

func (rc *RedisClient) GetClient() DbRedisDriver {
	return rc.Client
}

func (rc *RedisClient) Publish(channel string, message interface{}) (int64, error) {
	return rc.Client.Publish(channel, message)
}

func (rc *RedisClient) Connect() error {
	if rc.DbServer.Host == "" || rc.DbServer.Port == 0 {
		return errors.New("db server host or port is empty")
	}
	rc.Client = redis.New()
	err := rc.Client.Connect(rc.DbServer.Host, rc.DbServer.Port)
	return err
}

func (rc *RedisClient) Subscribe(cn chan []string, channel ...string) error {
	return rc.Client.Subscribe(cn, channel...)
}

func (rc *RedisClient) Close() error {
	return rc.Client.Close()
}

func (rc RedisClient) NewDBConnection() Client {
	err := rc.Connect()
	for err != nil {
		errors.Wrap(err, "db connection has failed")
		time.Sleep(time.Second)
		err = rc.Connect()
	}
	var dbClient Client = &rc
	return dbClient
}

func PublishWS(req entities.Request, roomID string, worker Client) {
	pubReq, err := json.Marshal(req)
	for err != nil {
		errors.Wrap(err, "req marshalling has failed")
	}

	worker.Connect()
	defer worker.Close()

	_, err = worker.Publish(roomID, pubReq)
	if err != nil {
		errors.Wrap(err, "publishing has failed")
	}
}

func (rc *RedisClient) GetAllDevices() []entities.DevData {
	rc.NewDBConnection()

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

		device.Meta.Type = key[1]
		device.Meta.Name = key[2]
		device.Meta.MAC = key[3]
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
	return devices
}

func (rc *RedisClient) GetKeyForConfig(mac string) string {
	return mac + partKeyToConfig
}

func (rc *RedisClient) Multi() (string, error) {
	return rc.Client.Multi()
}
func (rc *RedisClient) Discard() (string, error) {
	return rc.Client.Discard()
}
func (rc *RedisClient) Exec() ([]interface{}, error) {
	return rc.Client.Exec()
}

func (rc *RedisClient) ZRem(key string, arguments ...interface{}) (int64, error) {
	return rc.Client.ZRem(key, arguments)
}

func (rc *RedisClient) ZRange(key string, values ...interface{}) ([]string, error) {
	return rc.Client.ZRange(key, values)
}

func (rc *RedisClient) ZScore(key string, member interface{}) (int64, error) {
	return rc.Client.ZScore(key, member)
}
