package storages

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
	"github.com/Sirupsen/logrus"
)

func (rds *RedisDevStorage) getFridgeData(m *entities.DevMeta) (*entities.DevData, error) {
	devKey := "device:" + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + ":" + "params"

	dd := entities.DevData{
		Meta: *m,
		Data: make(map[string][]string),
	}

	params, err := rds.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeData(): SMembers() has failed")
	}

	data := make([][]string, len(params))
	for i, p := range params {
		data[i], err = rds.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: getFridgeData(): ZRangeByScore() has failed")
		}
		dd.Data[p] = data[i]
	}

	return &dd, err
}

func (rds *RedisDevStorage) setFridgeData(r *entities.Request) error {
	var fd entities.FridgeData
	err := json.Unmarshal([]byte(r.Data), &fd)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeData(): FridgeData unmarshalling has failed")
		return err
	}

	devKey := "device" + ":" + r.Meta.Type + ":" + r.Meta.Name + ":" + r.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	if _, err := rds.Client.SAdd("devParamsKeys", devParamsKey); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeData(): SAdd() has failed")
		rds.Client.Discard()
		return err
	}

	if _, err := rds.Client.HMSet(devKey, "ReqTime", r.Time); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeData(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}

	if _, err := rds.Client.SAdd(devParamsKey, "TempCam1", "TempCam2"); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeData(): SAdd() has failed")
		rds.Client.Discard()
		return err
	}

	if err := rds.setFridgeCameraData(fd.TempCam1, devParamsKey+":"+"TempCam1"); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeCameraData(): Multi() has failed")
		rds.Client.Discard()
		return err
	}

	if err := rds.setFridgeCameraData(fd.TempCam2, devParamsKey+":"+"TempCam2"); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeData(): setFridgeCameraData() has failed")
		rds.Client.Discard()
		return err
	}

	return nil
}

func (rds *RedisDevStorage) setFridgeCameraData(tempCam map[int64]float32, key string) error {
	for time, temp := range tempCam {
		_, err := rds.Client.ZAdd(key, strconv.FormatInt(int64(time), 10),
			strconv.FormatInt(int64(time), 10)+":"+strconv.FormatFloat(float64(temp), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "RedisDevStorage: setFridgeCameraData(): ZAdd() has failed")
			return err
		}
	}

	return nil
}

func (rds *RedisDevStorage) getFridgeConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	configKey := m.MAC + ":config"

	// get data from storage as []string
	to, err := rds.Client.HMGet(configKey, "TurnedOn")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): TurnedOn field extraction has failed")
		return nil, err
	}
	cf, err := rds.Client.HMGet(configKey, "CollectFreq")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): CollectFreq field extraction has failed")
		return nil, err
	}
	sf, err := rds.Client.HMGet(configKey, "SendFreq")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): SendFreq field extraction has failed")
		return nil, err
	}
	so, err := rds.Client.HMGet(configKey, "StreamOn")
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): StreamOn field extraction has failed")
		return nil, err
	}

	// parse obtained data to appropriate types
	pto, err := strconv.ParseBool(strings.Join(to, " "))
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): TurnedOn field parsing has failed")
		return nil, err
	}
	pcf, err := strconv.ParseInt(strings.Join(cf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): CollectFreq field parsing has failed")
		return nil, err
	}
	psf, err := strconv.ParseInt(strings.Join(sf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): SendFreq field parsing has failed")
		return nil, err
	}
	pso, err := strconv.ParseBool(strings.Join(so, " "))
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): StreamOn field parsing has failed")
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
		errors.Wrap(err, "RedisDevStorage: getFridgeConfig(): FridgeConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, err
}

func (rds *RedisDevStorage) setFridgeConfig(c *entities.DevConfig, m *entities.DevMeta) error {
	var fc entities.FridgeConfig
	if err := json.Unmarshal(c.Data, &fc); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): FridgeConfig unmarshalling has failed")
		return err
	}

	logrus.Info("New config: ", fc)

	config, _ := rds.getFridgeConfig(m)
	err := json.Unmarshal(c.Data, &config)

	logrus.Infof("Updated config: %s", config)

	configKey := c.MAC + ":config"
	if _, err := rds.Client.Multi(); err != nil{
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): Multi() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "TurnedOn", fc.TurnedOn); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "CollectFreq", fc.CollectFreq); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "SendFreq", fc.SendFreq); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	if _, err := rds.Client.HMSet(configKey, "StreamOn", fc.StreamOn); err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): HMSet() has failed")
		rds.Client.Discard()
		return err
	}
	_, err = rds.Client.Exec()
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setFridgeConfig(): Exec() has failed")
		rds.Client.Discard()
		return err
	}

	return nil
}

func (rds *RedisDevStorage) getFridgeDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	fc := entities.FridgeConfig{
		TurnedOn:    true,
		StreamOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000,
	}

	b, err := json.Marshal(fc)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: getFridgeDefaultConfig(): FridgeConfig marshalling has failed")
		return nil, err
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
