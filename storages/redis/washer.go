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

func (rs *RedisStorage) getWasherData(m *entities.DevMeta) (*entities.DevData, error) {
	devID := "device:" + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devID + ":" + "params"

	var device entities.DevData

	params, err := rs.Client.SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "func SMembers has failed")
	}
	device.Meta = *m
	device.Data = make(map[string][]string)

	values := make([][]string, len(params))
	for i, p := range params {
		values[i], err = rs.Client.ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "func ZRangeByScore has failed")
		}
		device.Data[p] = values[i]
	}
	return &device, err
}

func (rs *RedisStorage) setWasherData(req *entities.Request) error {
	var w entities.WasherData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.NewDecoder(bytes.NewBuffer(req.Data)).Decode(&w)
	if err != nil {
		errors.Wrap(err, "washer's DevData decoding has failed")
		return err
	}

	rs.Client.Multi()
	err = rs.setTurnoversData(w.Turnovers, devParamsKey+":"+"Turnovers")
	err = rs.setWaterTempData(w.WaterTemp, devParamsKey+":"+"WaterTemp")
	_, err = rs.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rs.Client.Discard()
		return err
	}

	return nil
}

func (rs *RedisStorage) setWaterTempData(TempCam map[int64]float32, key string) error {
	for t, v := range TempCam {
		rs.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	return nil
}

func (rs *RedisStorage) setTurnoversData(TempCam map[int64]int64, key string) error {
	for t, v := range TempCam {
		rs.Client.ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
	}
	return nil
}

func (rs *RedisStorage) getWasherConfig(mac string) (*entities.DevConfig, error) {
	return &entities.DevConfig{}, nil
}

func (rs *RedisStorage) setWasherConfig(mac string, c *entities.DevConfig) error {
	var timerMode *entities.TimerMode
	json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&timerMode)
	config := mac + ":config"
	rs.Client.ZAdd(config, timerMode.StartTime, timerMode.Name)
	return nil
}

func (rs *RedisStorage) getWasherDefaultConfig(m *entities.DevMeta) (*entities.DevConfig, error) {
	b, _ := json.Marshal(entities.WasherConfig{})
	return &entities.DevConfig{Data: b}, nil
}

func (rs *RedisStorage) sendWasherDefaultConfig(c *net.Conn, req *entities.Request) ([]byte, error) {
	var config *entities.DevConfig
	var err error
	configKey := req.Meta.MAC + ":" + "config"
	if ok, _ := rs.Client.Exists(configKey); ok {
		t := time.Now().UnixNano() / int64(time.Minute)
		config, err = rs.getActualWasherConfig(configKey, req.Meta.MAC, t, &req.Meta)
		if err != nil {
			errors.Wrap(err, "func getActualConfig has failed")
		}
	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config, err = rs.getWasherDefaultConfig(&req.Meta)
		if err != nil {
			errors.Wrap(err, "failed to get washer's default config")
		}
		rs.saveWasherToBD(configKey, config, req)
	}
	return config.Data, err
}

func (rs *RedisStorage) PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta entities.DevMeta) error {
	return nil
}

func (rs *RedisStorage) getActualWasherConfig(configInfo string, mac string, unixTime int64, m *entities.DevMeta) (*entities.DevConfig, error) {
	config, err := rs.getWasherDefaultConfig(m)
	config.MAC = mac

	mode, err := rs.Client.ZRangeByScore(configInfo, unixTime-100, unixTime+100)
	if err != nil {
		errors.Wrap(err, "func ZRangeByScore has failed")
	}

	if len(mode) == 0 {
		return config, err
	}

	configWasher := entities.LightMode
	config.Data, err = json.Marshal(configWasher)
	if err != nil {
		errors.Wrap(err, "washer's DevConfig marshalling has failed")
	}
	return config, err
}

func (rs *RedisStorage) saveWasherToBD(configInfo string, config *entities.DevConfig, req *entities.Request) {
	var timerMode entities.TimerMode
	json.NewDecoder(bytes.NewBuffer(config.Data)).Decode(&timerMode)

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	rs.Client.Multi()
	rs.Client.SAdd("devParamsKeys", devParamsKey)
	rs.Client.HMSet(devKey, "ReqTime", req.Time)
	rs.Client.SAdd(devParamsKey, "Turnovers", "WaterTemp")
	rs.Client.ZAdd(configInfo, timerMode.StartTime, timerMode.Name)
	_, err := rs.Client.Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		rs.Client.Discard()
	}
}
