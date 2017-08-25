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

func (rc *RedisStorage) getFridgeData(devParamsKey string, m *entities.DevMeta) (*entities.DevData, error) {
	var f entities.DevData

	params, err := rc.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "can't read members from devParamsKeys")
	}

	f.Meta = *m
	f.Data = make(map[string][]string)

	values := make([][]string, len(params))
	for i, p := range params {
		values[i], err = rc.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "func ZRangeByScore has failed")
		}

		f.Data[p] = values[i]
	}
	return &f, err
}

func (rc *RedisStorage) setFridgeData(req *entities.Request) error {
	var f entities.FridgeData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.Unmarshal([]byte(req.Data), &f)
	if err != nil {
		errors.New("Error in SetDevData")
		return err
	}

	rc.Client.Multi()
	rc.Client.SAdd("devParamsKeys", devParamsKey)
	rc.Client.HMSet(devKey, "ReqTime", req.Time)
	rc.Client.SAdd(devParamsKey, "TempCam1", "TempCam2")
	rc.setFridgeCameraData(f.TempCam1, devParamsKey+":"+"TempCam1")
	rc.setFridgeCameraData(f.TempCam2, devParamsKey+":"+"TempCam2")
	_, err = rc.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rc.Client.Discard()
		return err
	}

	return nil
}

func (rc *RedisStorage) setFridgeCameraData(tempCam map[int64]float32, key string) error {
	for t, v := range tempCam {
		rc.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	return nil
}

func (rc *RedisStorage) getFridgeConfig(configInfo string, mac string) (*entities.DevConfig, error) {
	state, err := rc.Client.HMGet(configInfo, "TurnedOn")
	if err != nil {
		errors.Wrap(err, "TurnedOn field extraction has failed")
	}

	sendFreq, err := rc.Client.HMGet(configInfo, "SendFreq")
	if err != nil {
		errors.Wrap(err, "SendFreq field extraction has failed")
	}

	collectFreq, err := rc.Client.HMGet(configInfo, "CollectFreq")
	if err != nil {
		errors.Wrap(err, "CollectFreq field extraction has failed")
	}

	streamOn, err := rc.Client.HMGet(configInfo, "StreamOn")
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

func (rc *RedisStorage) setFridgeConfig(configInfo string, c *entities.DevConfig) error {
	var fridgeConfig entities.FridgeConfig
	json.Unmarshal(c.Data, &fridgeConfig)

	rc.Client.Multi()
	rc.Client.HMSet(configInfo, "TurnedOn", fridgeConfig.TurnedOn)
	rc.Client.HMSet(configInfo, "CollectFreq", fridgeConfig.CollectFreq)
	rc.Client.HMSet(configInfo, "SendFreq", fridgeConfig.SendFreq)
	rc.Client.HMSet(configInfo, "StreamOn", fridgeConfig.StreamOn)
	_, err := rc.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rc.Client.Discard()
	}
	return nil
}

func (rc *RedisStorage) getFridgeDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
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

func (rc *RedisStorage) sendFridgeDefaultConfig(c *net.Conn, req *entities.Request) ([]byte, error) {
	var config *entities.DevConfig
	var err error
	configInfo := req.Meta.MAC + ":" + "config" // key
	if ok, _ := rc.Client.Exists(configInfo); ok {
		config, err = rc.getFridgeConfig(configInfo, req.Meta.MAC)
		if err != nil {
			errors.New("fridge config ")
		}
		log.Println("Old Device with MAC: ", req.Meta.MAC, "detected.")
	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config, err = rc.getFridgeDefaultConfig(&req.Meta)
		if err != nil {
			errors.Wrap(err, "fridge config extraction has failed")
		}
		rc.setFridgeConfig(configInfo, config)
	}

	return config.Data, err
}
