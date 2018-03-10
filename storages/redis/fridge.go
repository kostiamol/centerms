package storages

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/entities"
	"github.com/pkg/errors"
)

func (s *RedisStorage) getFridgeData(m *entities.DevMeta) (*entities.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	paramsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := redis.Strings(s.conn.Do("SMEMBERS", paramsKey))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeData(): SMembers() has failed")
	}

	for _, p := range params {
		data[p], err = redis.Strings(s.conn.Do("ZRANGEBYSCORE", paramsKey+":"+p, "-inf", "inf"))
		if err != nil {
			errors.Wrap(err, "RedisStorage: getFridgeData(): ZRangeByScore() has failed")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeData(): fridge data marshalling has failed")
		return nil, err
	}

	return &entities.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (s *RedisStorage) saveFridgeData(data *entities.DevData) error {
	var fd entities.FridgeData
	if err := json.Unmarshal([]byte(data.Data), &fd); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): FridgeData unmarshalling has failed")
		return err
	}

	devKey := partialDevKey + data.Meta.Type + ":" + data.Meta.Name + ":" + data.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := s.conn.Do("HMSET", "devParamsKeys", paramsKey); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): SAdd() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("HMSET", devKey, "ReqTime", data.Time); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): HMSet() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("SADD", paramsKey, "TopCompart", "BotCompart"); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): SAdd() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if err := s.setFridgeCameraData(fd.TopCompart, paramsKey+":"+"TopCompart"); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeCameraData(): Multi() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if err := s.setFridgeCameraData(fd.BotCompart, paramsKey+":"+"BotCompart"); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): setFridgeCameraData() has failed")
		s.conn.Do("DISCARD")
		return err
	}

	return nil
}

func (s *RedisStorage) setFridgeCameraData(tempCam map[int64]float32, key string) error {
	for time, temp := range tempCam {
		_, err := s.conn.Do("ZADD", key, strconv.FormatInt(int64(time), 10),
			strconv.FormatInt(int64(time), 10)+":"+strconv.FormatFloat(float64(temp), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "RedisStorage: setFridgeCameraData(): ZAdd() has failed")
			return err
		}
	}

	return nil
}

func (s *RedisStorage) getFridgeConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	configKey := m.MAC + partialDevConfigKey

	to, err := redis.Strings(s.conn.Do("HMGET", configKey, "TurnedOn"))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): TurnedOn field extraction has failed")
		return nil, err
	}
	cf, err := redis.Strings(s.conn.Do("HMGET", configKey, "CollectFreq"))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): CollectFreq field extraction has failed")
		return nil, err
	}
	sf, err := redis.Strings(s.conn.Do("HMGET", configKey, "SendFreq"))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): SendFreq field extraction has failed")
		return nil, err
	}

	pto, err := strconv.ParseBool(strings.Join(to, " "))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): TurnedOn field parsing has failed")
		return nil, err
	}

	pcf, err := strconv.ParseInt(strings.Join(cf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): CollectFreq field parsing has failed")
		return nil, err
	}
	psf, err := strconv.ParseInt(strings.Join(sf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): SendFreq field parsing has failed")
		return nil, err
	}

	config := entities.FridgeConfig{
		TurnedOn:    pto,
		CollectFreq: pcf,
		SendFreq:    psf,
	}

	b, err := json.Marshal(&config)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): FridgeConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, err
}

func (s *RedisStorage) setFridgeConfig(c *entities.DevConfig, m *entities.DevMeta) error {
	var config *entities.DevConfig
	if ok, err := s.DevIsRegistered(m); ok {
		if err != nil {
			errors.Wrapf(err, "RedisStorage: setFridgeConfig(): DevIsRegistered() has failed")
			return err
		}
		if config, err = s.getFridgeConfig(m); err != nil {
			return err
		}
	} else {
		if err != nil {
			errors.Wrapf(err, "RedisStorage: setFridgeConfig(): DevIsRegistered() has failed")
			return err
		}
		if config, err = s.getFridgeDefaultConfig(m); err != nil {
			return err
		}
	}

	var fc entities.FridgeConfig
	if err := json.Unmarshal(config.Data, &fc); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Unmarshal() has failed")
		return err
	}

	if err := json.Unmarshal(c.Data, &fc); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Unmarshal() has failed")
		return err
	}

	configKey := c.MAC + partialDevConfigKey
	if _, err := s.conn.Do("MULTI"); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Multi() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("HMSET", configKey, "TurnedOn", fc.TurnedOn); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("HMSET", configKey, "CollectFreq", fc.CollectFreq); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("HMSET", configKey, "SendFreq", fc.SendFreq); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		s.conn.Do("DISCARD")
		return err
	}

	_, err := s.conn.Do("EXEC")
	if err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Exec() has failed")
		s.conn.Do("DISCARD")
		return err
	}

	return nil
}

func (s *RedisStorage) getFridgeDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	b, err := json.Marshal(entities.DefaultFridgeConfig)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeDefaultConfig(): FridgeConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
