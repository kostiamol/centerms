package storages

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

func (rs *RedisStorage) getFridgeData(m *entities.DevMeta) (*entities.DevData, error) {
	devID := "device:" + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devID + ":" + "params"

	var f entities.DevData

	params, err := rs.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "can't read members from devParamsKeys")
	}

	f.Meta = *m
	f.Data = make(map[string][]string)

	values := make([][]string, len(params))
	for i, p := range params {
		values[i], err = rs.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "func ZRangeByScore has failed")
		}

		f.Data[p] = values[i]
	}
	return &f, err
}

func (rs *RedisStorage) setFridgeData(req *entities.Request) error {
	var f entities.FridgeData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.Unmarshal([]byte(req.Data), &f)
	if err != nil {
		errors.New("Error in SetDevData")
		return err
	}

	rs.Client.Multi()
	rs.Client.SAdd("devParamsKeys", devParamsKey)
	rs.Client.HMSet(devKey, "ReqTime", req.Time)
	rs.Client.SAdd(devParamsKey, "TempCam1", "TempCam2")
	rs.setFridgeCameraData(f.TempCam1, devParamsKey+":"+"TempCam1")
	rs.setFridgeCameraData(f.TempCam2, devParamsKey+":"+"TempCam2")
	_, err = rs.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rs.Client.Discard()
		return err
	}

	return nil
}

func (rs *RedisStorage) setFridgeCameraData(tempCam map[int64]float32, key string) error {
	for t, v := range tempCam {
		rs.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	return nil
}

func (rs *RedisStorage) getFridgeConfig(mac string) (*entities.DevConfig, error) {
	config := mac + ":config"
	state, err := rs.Client.HMGet(config, "TurnedOn")
	if err != nil {
		errors.Wrap(err, "TurnedOn field extraction has failed")
	}

	sendFreq, err := rs.Client.HMGet(config, "SendFreq")
	if err != nil {
		errors.Wrap(err, "SendFreq field extraction has failed")
	}

	collectFreq, err := rs.Client.HMGet(config, "CollectFreq")
	if err != nil {
		errors.Wrap(err, "CollectFreq field extraction has failed")
	}

	streamOn, err := rs.Client.HMGet(config, "StreamOn")
	if err != nil {
		errors.Wrap(err, "StreamOn field extraction has failed")
	}

	stateBool, _ := strconv.ParseBool(strings.Join(state, " "))
	sendFreqInt, _ := strconv.Atoi(strings.Join(sendFreq, " "))
	collectFreqInt, _ := strconv.Atoi(strings.Join(collectFreq, " "))
	streamOnBool, _ := strconv.ParseBool(strings.Join(streamOn, " "))

	fc := entities.FridgeConfig{
		TurnedOn:    stateBool,
		CollectFreq: int64(collectFreqInt),
		SendFreq:    int64(sendFreqInt),
		StreamOn:    streamOnBool,
	}

	var devConfig entities.DevConfig

	arrByte, err := json.Marshal(&fc)
	if err != nil {
		errors.Wrap(err, "fridgeConfig marshalling has failed")
	}

	devConfig = entities.DevConfig{
		MAC:  mac,
		Data: arrByte,
	}

	log.Println("Configuration from DB: ", fc.TurnedOn, fc.SendFreq, fc.CollectFreq)
	return &devConfig, err
}

func (rs *RedisStorage) setFridgeConfig(mac string, c *entities.DevConfig) error {
	var fridgeConfig entities.FridgeConfig
	json.Unmarshal(c.Data, &fridgeConfig)
	config := mac + ":config"
	rs.Client.Multi()
	rs.Client.HMSet(config, "TurnedOn", fridgeConfig.TurnedOn)
	rs.Client.HMSet(config, "CollectFreq", fridgeConfig.CollectFreq)
	rs.Client.HMSet(config, "SendFreq", fridgeConfig.SendFreq)
	rs.Client.HMSet(config, "StreamOn", fridgeConfig.StreamOn)
	_, err := rs.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rs.Client.Discard()
	}
	return nil
}

func (rs *RedisStorage) getFridgeDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	config := entities.FridgeConfig{
		TurnedOn:    true,
		StreamOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000,
	}

	arrByte, err := json.Marshal(config)
	if err != nil {
		errors.Wrap(err, "FridgeConfig marshalling has failed")
	}

	return &entities.DevConfig{
		MAC:  m.MAC,
		Data: arrByte,
	}, err
}

func (rs *RedisStorage) sendFridgeDefaultConfig(c *net.Conn, req *entities.Request) ([]byte, error) {
	var config *entities.DevConfig
	var err error
	configKey := req.Meta.MAC + ":" + "config"
	if ok, _ := rs.Client.Exists(configKey); ok {
		config, err = rs.getFridgeConfig(req.Meta.MAC)
		if err != nil {
			errors.New("fridge config ")
		}
	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config, err = rs.getFridgeDefaultConfig(&req.Meta)
		if err != nil {
			errors.Wrap(err, "fridge config extraction has failed")
		}
		rs.setFridgeConfig(configKey, config)
	}

	return config.Data, err
}
























