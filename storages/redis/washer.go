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

func (washer *Washer) GetDevData(devParamsKey string, devMeta entities.DevMeta, client db.Client) entities.DevData {
	var device entities.DevData

	params, err := client.GetClient().SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "func SMembers has failed")
	}
	device.Meta = devMeta
	device.Data = make(map[string][]string)

	values := make([][]string, len(params))
	for i, p := range params {
		values[i], err = client.GetClient().ZRangeByScore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			errors.Wrap(err, "func ZRangeByScore has failed")
		}
		device.Data[p] = values[i]
	}
	return device
}

func (washer *Washer) SetDevData(req *entities.Request, client db.Client) *entities.ServerError {

	var devData WasherData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.NewDecoder(bytes.NewBuffer(req.Data)).Decode(&devData)
	if err != nil {
		errors.Wrap(err, "washer's DevData decoding has failed")
		return &entities.ServerError{Error: err}
	}

	client.GetClient().Multi()
	err = setTurnoversData(devData.Turnovers, devParamsKey+":"+"Turnovers", client)
	err = setWaterTempData(devData.WaterTemp, devParamsKey+":"+"WaterTemp", client)
	_, err = client.GetClient().Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		client.GetClient().Discard()
		return &entities.ServerError{Error: err}
	}

	return nil
}

func setWaterTempData(TempCam map[int64]float32, key string, client db.Client) error {
	for t, v := range TempCam {
		client.GetClient().ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))

	}
	return nil
}

func setTurnoversData(TempCam map[int64]int64, key string, client db.Client) error {
	for t, v := range TempCam {
		client.GetClient().ZAdd(key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
	}
	return nil
}

func (washer *Washer) GetDevConfig(configInfo, mac string, client db.Client) *entities.DevConfig {
	return &entities.DevConfig{}
}

func (washer *Washer) SetDevConfig(configInfo string, config *entities.DevConfig, client db.Client) {
	var timerMode *TimerMode
	json.NewDecoder(bytes.NewBuffer(config.Data)).Decode(&timerMode)
	client.GetClient().ZAdd(configInfo, timerMode.StartTime, timerMode.Name)
}

func (washer *Washer) GetDefaultConfig() *entities.DevConfig {
	b, _ := json.Marshal(WasherConfig{})
	return &entities.DevConfig{Data: b}
}

func (washer *Washer) SendDefaultConfigurationTCP(conn net.Conn, dbClient db.Client, req *entities.Request) []byte {
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

func (washer *Washer) PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, meta entities.DevMeta, client db.Client) {

}

func (washer *Washer) getActualConfig(configInfo, mac string, client db.Client, unixTime int64) (*entities.DevConfig, error) {
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

func (washer *Washer) saveDeviceToBD(configInfo string, config *entities.DevConfig, client db.Client, req *entities.Request) {
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
