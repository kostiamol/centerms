package storages

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

func (rc *RedisStorage) getWasherData(devParamsKey string, m *entities.DevMeta) entities.DevData {
	var device entities.DevData

	params, err := rc.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "func SMembers has failed")
	}
	device.Meta = *m
	device.Data = make(map[string][]string)

	values := make([][]string, len(params))
	for i, p := range params {
		values[i], err = rc.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "func ZRangeByScore has failed")
		}
		device.Data[p] = values[i]
	}
	return device
}

func (rc *RedisStorage) setWasherData(req *entities.Request) error {
	var w entities.WasherData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.NewDecoder(bytes.NewBuffer(req.Data)).Decode(&w)
	if err != nil {
		errors.Wrap(err, "washer's DevData decoding has failed")
		return err
	}

	rc.Client.Multi()
	err = setTurnoversData(w.Turnovers, devParamsKey+":"+"Turnovers")
	err = setWaterTempData(w.WaterTemp, devParamsKey+":"+"WaterTemp")
	_, err = rc.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rc.Client.Discard()
		return err
	}

	return nil
}

func setWaterTempData(TempCam map[int64]float32, key string) error {
	for t, v := range TempCam {
		client.GetClient().ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	return nil
}

func setTurnoversData(TempCam map[int64]int64, key string) error {
	for t, v := range TempCam {
		client.GetClient().ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
	}
	return nil
}

func (rc *RedisStorage) GetDevConfig(configInfo, mac string) *entities.DevConfig {
	return &entities.DevConfig{}
}

func (rc *RedisStorage) SetDevConfig(configInfo string, config *entities.DevConfig) {
	var timerMode *TimerMode
	json.NewDecoder(bytes.NewBuffer(config.Data)).Decode(&timerMode)
	client.GetClient().ZAdd(configInfo, timerMode.StartTime, timerMode.Name)
}

func (rc *RedisStorage) GetDefaultConfig() *entities.DevConfig {
	b, _ := json.Marshal(WasherConfig{})
	return &entities.DevConfig{Data: b}
}

func (rc *RedisStorage) SendDefaultConfigurationTCP(conn net.Conn, dbClient db.Client, req *entities.Request) []byte {
	var config *entities.DevConfig
	var err error
	configInfo := req.Meta.MAC + ":" + "config" // key

	if ok, _ := dbClient.GetClient().Exists(configInfo); ok {
		t := time.Now().UnixNano() / int64(time.Minute)
		config, err = washer.getActualConfig(configInfo, req.Meta.MAC, dbClient, t)
		if err != nil {
			errors.Wrap(err, "func getActualConfig has failed")
		}

	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config = washer.GetDefaultConfig()
		washer.saveDeviceToBD(configInfo, config, dbClient, req)
	}
	return config.Data
}

func (rc *RedisStorage) PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta entities.DevMeta, client db.Client) {

}

func (rc *RedisStorage) getActualConfig(configInfo, mac string, client db.Client, unixTime int64) (*entities.DevConfig, error) {
	config := washer.GetDefaultConfig()
	config.MAC = mac

	mode, err := client.GetClient().ZRangeByScore(configInfo, unixTime-100, unixTime+100)
	if err != nil {
		errors.Wrap(err, "func ZRangeByScore has failed")
	}

	if len(mode) == 0 {
		return config, err
	}

	configWasher := washer.selectMode(mode[0])
	config.Data, err = json.Marshal(configWasher)
	if err != nil {
		errors.Wrap(err, "washer's DevConfig marshalling has failed")
	}
	return config, err
}

func (rc *RedisStorage) saveDeviceToBD(configInfo string, config *entities.DevConfig, client db.Client, req *entities.Request) {
	var timerMode TimerMode
	json.NewDecoder(bytes.NewBuffer(config.Data)).Decode(&timerMode)

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	client.GetClient().Multi()
	client.GetClient().SAdd("devParamsKeys", devParamsKey)
	client.GetClient().HMSet(devKey, "ReqTime", req.Time)
	client.GetClient().SAdd(devParamsKey, "Turnovers", "WaterTemp")
	client.GetClient().ZAdd(configInfo, timerMode.StartTime, timerMode.Name)
	_, err := client.GetClient().Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		client.GetClient().Discard()
	}
}
