package storages

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

func (rds *RedisStorage) getWasherData(m *entities.DevMeta) (*entities.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + partialDevParamsKey

	dd := entities.DevData{
		Meta: *m,
		Data: make(map[string][]string),
	}

	params, err := rds.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherData(): can't read members from devParamsKeys")
	}

	data := make([][]string, len(params))
	for i, p := range params {
		data[i], err = rds.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: getWasherData(): can't read members from sorted set")
		}
		dd.Data[p] = data[i]
	}

	return &dd, err
}

func (rds *RedisStorage) saveWasherData(r *entities.SaveDevDataRequest) error {
	var wd entities.WasherData
	if err := json.NewDecoder(bytes.NewBuffer(r.Data)).Decode(&wd); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): WasherData decoding has failed")
		return err
	}

	devKey := partialDevKey + r.Meta.Type + ":" + r.Meta.Name + ":" + r.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := rds.Client.Multi(); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): Multi() has failed")
		rds.Client.Discard()
		return err
	}
	if err := rds.setTurnoversData(wd.Turnovers, paramsKey+":"+"Turnovers"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): setTurnoversData() has failed")
		rds.Client.Discard()
		return err
	}
	if err := rds.setWaterTempData(wd.WaterTemp, paramsKey+":"+"WaterTemp"); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): setWaterTempData() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.Exec(); err != nil {
		errors.Wrap(err, "RedisDevStorage: saveWasherData(): Exec() has failed")
		rds.Client.Discard()
		return err
	}

	return nil
}

func (rds *RedisStorage) setTurnoversData(TempCam map[int64]int64, key string) error {
	for t, v := range TempCam {
		_, err := rds.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: setTurnoversData(): adding to sorted set has failed")
			return err
		}
	}

	return nil
}

func (rds *RedisStorage) setWaterTempData(TempCam map[int64]float32, key string) error {
	for t, v := range TempCam {
		_, err := rds.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: setWaterTempData(): adding to sorted set has failed")
			return err
		}
	}

	return nil
}

func (rds *RedisStorage) getWasherConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	config, err := rds.getWasherDefaultConfig(m)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getWasherConfig(): getWasherDefaultConfig() has failed")
	}

	config.MAC = m.MAC
	configKey := m.MAC + partialDevConfigKey
	unixTime:= int64(100) // fake
	mode, err := rds.Client.ZRangeByScore(configKey, unixTime-100, unixTime+100)
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

func (rds *RedisStorage) setWasherConfig(c *entities.DevConfig) error {
	var tm *entities.TimerMode
	err := json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&tm)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setWasherConfig(): Decode() has failed")
	}

	configKey := c.MAC + partialDevConfigKey
	rds.Client.ZAdd(configKey, tm.StartTime, tm.Name)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setWasherConfig(): ZAdd() has failed")
	}

	return err
}

func (rds *RedisStorage) getWasherDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
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
