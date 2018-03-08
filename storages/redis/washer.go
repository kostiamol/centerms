package storages

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/entities"
	"github.com/pkg/errors"
)

func (s *RedisStorage) getWasherData(m *entities.DevMeta) (*entities.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + partialDevParamsKey

	devData := entities.DevData{
		Meta: *m,
		Data: make(map[string][]string),
	}

	params, err := redis.Strings(s.conn.Do("SMEMBERS", devParamsKey))
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherData(): can't read members from devParamsKeys")
	}

	data := make([][]string, len(params))
	for i, p := range params {
		data[i], err = redis.Strings(s.conn.Do("ZRANGEBYSCORE", devParamsKey+":"+p, "-inf", "inf"))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: getWasherData(): can't read members from sorted set")
		}
		devData.Data[p] = data[i]
	}

	return &devData, err
}

func (s *RedisStorage) saveWasherData(data *entities.RawDevData) error {
	var wd entities.WasherData
	if err := json.NewDecoder(bytes.NewBuffer(data.Data)).Decode(&wd); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): WasherData decoding has failed")
		return err
	}

	devKey := partialDevKey + data.Meta.Type + ":" + data.Meta.Name + ":" + data.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := s.conn.Do("MULTI"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): Multi() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if err := s.setTurnoversData(wd.Turnovers, paramsKey+":"+"Turnovers"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): setTurnoversData() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if err := s.setWaterTempData(wd.WaterTemp, paramsKey+":"+"WaterTemp"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): setWaterTempData() has failed")
		s.conn.Do("DISCARD")
		return err
	}
	if _, err := s.conn.Do("EXEC"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): Exec() has failed")
		s.conn.Do("DISCARD")
		return err
	}

	return nil
}

func (s *RedisStorage) setTurnoversData(TempCam map[int64]int64, key string) error {
	for t, v := range TempCam {
		_, err := s.conn.Do("ZADD", key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: setTurnoversData(): adding to sorted set has failed")
			return err
		}
	}

	return nil
}

func (s *RedisStorage) setWaterTempData(TempCam map[int64]float32, key string) error {
	for t, v := range TempCam {
		_, err := s.conn.Do("ZADD", key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: setWaterTempData(): adding to sorted set has failed")
			return err
		}
	}

	return nil
}

func (s *RedisStorage) getWasherConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	config, err := s.getWasherDefaultConfig(m)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherConfig(): getWasherDefaultConfig() has failed")
	}

	config.MAC = m.MAC
	configKey := m.MAC + partialDevConfigKey
	unixTime := int64(100) // fake
	mode, err := redis.Strings(s.conn.Do("ZRANGEBYSCORE", configKey, unixTime-100, unixTime+100))
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherConfig(): ZRangeByScore() has failed")
	}
	if len(mode) == 0 {
		return config, err
	}

	configWasher := entities.LightMode
	config.Data, err = json.Marshal(configWasher)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherConfig(): WasherConfig marshalling has failed")
	}

	return config, err
}

func (s *RedisStorage) setWasherConfig(c *entities.DevConfig) error {
	var tm *entities.TimerMode
	err := json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&tm)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setWasherConfig(): Decode() has failed")
	}

	configKey := c.MAC + partialDevConfigKey
	_, err = s.conn.Do("ZADD", configKey, tm.StartTime, tm.Name)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setWasherConfig(): ZAdd() has failed")
	}

	return err
}

func (s *RedisStorage) getWasherDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	b, err := json.Marshal(entities.StandardMode)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherDefaultConfig(): WasherConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
