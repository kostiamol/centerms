package storages

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

func (rds *RedisStorage) getFridgeData(m *entities.DevMeta) (*entities.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	paramsKey := devKey + partialDevParamsKey

	dd := entities.DevData{
		Meta: *m,
		Data: make(map[string][]string),
	}

	params, err := rds.Client.SMembers(paramsKey)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeData(): SMembers() has failed")
	}

	data := make([][]string, len(params))
	for i, p := range params {
		data[i], err = rds.Client.ZRangeByScore(paramsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "RedisStorage: getFridgeData(): ZRangeByScore() has failed")
		}
		dd.Data[p] = data[i]
	}

	return &dd, err
}

func (rds *RedisStorage) saveFridgeData(r *entities.Request) error {
	var fd entities.FridgeData
	if err := json.Unmarshal([]byte(r.Data), &fd); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): FridgeData unmarshalling has failed")
		return err
	}

	devKey := partialDevKey + r.Meta.Type + ":" + r.Meta.Name + ":" + r.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := rds.Client.SAdd("devParamsKeys", paramsKey); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): SAdd() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(devKey, "ReqTime", r.Time); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.SAdd(paramsKey, "TempCam1", "TempCam2"); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): SAdd() has failed")
		rds.Client.Discard()
		return err
	}
	if err := rds.setFridgeCameraData(fd.TempCam1, paramsKey+":"+"TempCam1"); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeCameraData(): Multi() has failed")
		rds.Client.Discard()
		return err
	}
	if err := rds.setFridgeCameraData(fd.TempCam2, paramsKey+":"+"TempCam2"); err != nil {
		errors.Wrap(err, "RedisStorage: saveFridgeData(): setFridgeCameraData() has failed")
		rds.Client.Discard()
		return err
	}

	return nil
}

func (rds *RedisStorage) setFridgeCameraData(tempCam map[int64]float32, key string) error {
	for time, temp := range tempCam {
		_, err := rds.Client.ZAdd(key, strconv.FormatInt(int64(time), 10),
			strconv.FormatInt(int64(time), 10)+":"+strconv.FormatFloat(float64(temp), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "RedisStorage: setFridgeCameraData(): ZAdd() has failed")
			return err
		}
	}

	return nil
}

func (rds *RedisStorage) getFridgeConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	configKey := m.MAC + partialDevConfigKey

	to, err := rds.Client.HMGet(configKey, "TurnedOn")
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): TurnedOn field extraction has failed")
		return nil, err
	}
	cf, err := rds.Client.HMGet(configKey, "CollectFreq")
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): CollectFreq field extraction has failed")
		return nil, err
	}
	sf, err := rds.Client.HMGet(configKey, "SendFreq")
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): SendFreq field extraction has failed")
		return nil, err
	}
	so, err := rds.Client.HMGet(configKey, "StreamOn")
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): StreamOn field extraction has failed")
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
	pso, err := strconv.ParseBool(strings.Join(so, " "))
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): StreamOn field parsing has failed")
		return nil, err
	}

	fc := entities.FridgeConfig{
		TurnedOn:    pto,
		CollectFreq: pcf,
		SendFreq:    psf,
		StreamOn:    pso,
	}

	b, err := json.Marshal(&fc)
	if err != nil {
		errors.Wrap(err, "RedisStorage: getFridgeConfig(): FridgeConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, err
}

func (rds *RedisStorage) setFridgeConfig(c *entities.DevConfig, m *entities.DevMeta) error {
	var cfc *entities.DevConfig
	if ok, err := rds.DevIsRegistered(m); ok {
		if err != nil {
			errors.Wrapf(err,"RedisStorage: setFridgeConfig(): DevIsRegistered() has failed")
			return err
		}
		cfc, err = rds.getFridgeConfig(m)
	} else {
		if err != nil {
			errors.Wrapf(err,"RedisStorage: setFridgeConfig(): DevIsRegistered() has failed")
			return err
		}
		cfc, err = rds.getFridgeDefaultConfig(m)
	}

	var fc entities.FridgeConfig
	if err := json.Unmarshal(cfc.Data, &fc); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Unmarshal() has failed")
		return err
	}

	if err := json.Unmarshal(c.Data, &fc); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Unmarshal() has failed")
		return err
	}

	configKey := c.MAC + partialDevConfigKey
	if _, err := rds.Client.Multi(); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Multi() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "TurnedOn", fc.TurnedOn); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "CollectFreq", fc.CollectFreq); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "SendFreq", fc.SendFreq); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "StreamOn", fc.StreamOn); err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	_, err := rds.Client.Exec()
	if err != nil {
		errors.Wrap(err, "RedisStorage: setFridgeConfig(): Exec() has failed")
		rds.Client.Discard()
		return err
	}

	return nil
}

func (rds *RedisStorage) getFridgeDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
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
