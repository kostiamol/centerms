package devices

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/entities"
	"github.com/pkg/errors"
)

type Fridge struct {
	Data   FridgeData       `json:"data"`
	Config FridgeConfig     `json:"config"`
	Meta   entities.DevMeta `json:"meta"`
}
type FridgeData struct {
	TempCam1 map[int64]float32 `json:"tempCam1"`
	TempCam2 map[int64]float32 `json:"tempCam2"`
}

type FridgeConfig struct {
	TurnedOn    bool  `json:"turnedOn"`
	StreamOn    bool  `json:"streamOn"`
	CollectFreq int64 `json:"collectFreq"`
	SendFreq    int64 `json:"sendFreq"`
}

func (fridge *Fridge) GetDevData(devParamsKey string, devMeta entities.DevMeta, client db.Client) entities.DevData {
	var device entities.DevData

	params, err := client.GetClient().SMembers(devParamsKey)
	if err != nil {
		errors.Wrap(err, "can't read members from devParamsKeys")
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

func (fridge *Fridge) SetDevData(req *entities.Request, client db.Client) *entities.ServerError {

	var devData FridgeData

	devKey := "device" + ":" + req.Meta.Type + ":" + req.Meta.Name + ":" + req.Meta.MAC
	devParamsKey := devKey + ":" + "params"

	err := json.Unmarshal([]byte(req.Data), &devData)
	if err != nil {
		log.Error("Error in SetDevData")
		return &entities.ServerError{Error: err}
	}

	client.GetClient().Multi()
	client.GetClient().SAdd("devParamsKeys", devParamsKey)
	client.GetClient().HMSet(devKey, "ReqTime", req.Time)
	client.GetClient().SAdd(devParamsKey, "TempCam1", "TempCam2")
	setCameraData(devData.TempCam1, devParamsKey+":"+"TempCam1", client)
	setCameraData(devData.TempCam2, devParamsKey+":"+"TempCam2", client)
	_, err = client.GetClient().Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		client.GetClient().Discard()
		return &entities.ServerError{Error: err}
	}

	return nil
}

func setCameraData(TempCam map[int64]float32, key string, client db.Client) error {
	for time, value := range TempCam {
		client.GetClient().ZAdd(key, Int64ToString(time), Int64ToString(time)+":"+Float64ToString(value))
	}
	return nil
}

func (fridge *Fridge) GetDevConfig(configInfo, mac string, client db.Client) *entities.DevConfig {
	var fridgeConfig FridgeConfig = *fridge.getFridgeConfig(configInfo, mac, client)
	var devConfig entities.DevConfig

	arrByte, err := json.Marshal(&fridgeConfig)
	if err != nil {
		errors.Wrap(err, "fridgeConfig marshalling has failed")
	}

	devConfig = entities.DevConfig{
		MAC:  mac,
		Data: arrByte,
	}

	log.Println("Configuration from DB: ", fridgeConfig.TurnedOn, fridgeConfig.SendFreq, fridgeConfig.CollectFreq)
	return &devConfig
}

func (fridge *Fridge) getFridgeConfig(configInfo, mac string, client db.Client) *FridgeConfig {

	state, err := client.GetClient().HMGet(configInfo, "TurnedOn")
	if err != nil {
		errors.Wrap(err, "TurnedOn field extraction has failed")
	}

	sendFreq, err := client.GetClient().HMGet(configInfo, "SendFreq")
	if err != nil {
		errors.Wrap(err, "SendFreq field extraction has failed")
	}

	collectFreq, err := client.GetClient().HMGet(configInfo, "CollectFreq")
	if err != nil {
		errors.Wrap(err, "CollectFreq field extraction has failed")
	}

	streamOn, err := client.GetClient().HMGet(configInfo, "StreamOn")
	if err != nil {
		errors.Wrap(err, "StreamOn field extraction has failed")
	}

	stateBool, _ := strconv.ParseBool(strings.Join(state, " "))
	sendFreqInt, _ := strconv.Atoi(strings.Join(sendFreq, " "))
	collectFreqInt, _ := strconv.Atoi(strings.Join(collectFreq, " "))
	streamOnBool, _ := strconv.ParseBool(strings.Join(streamOn, " "))

	return &FridgeConfig{
		TurnedOn:    stateBool,
		CollectFreq: int64(collectFreqInt),
		SendFreq:    int64(sendFreqInt),
		StreamOn:    streamOnBool,
	}
}

func (fridge *Fridge) SetDevConfig(configInfo string, config *entities.DevConfig, client db.Client) {
	var fridgeConfig FridgeConfig
	json.Unmarshal(config.Data, &fridgeConfig)

	client.GetClient().Multi()
	client.GetClient().HMSet(configInfo, "TurnedOn", fridgeConfig.TurnedOn)
	client.GetClient().HMSet(configInfo, "CollectFreq", fridgeConfig.CollectFreq)
	client.GetClient().HMSet(configInfo, "SendFreq", fridgeConfig.SendFreq)
	client.GetClient().HMSet(configInfo, "StreamOn", fridgeConfig.StreamOn)
	_, err := client.GetClient().Exec()
	if err != nil {
		errors.Wrap(err, "trash")
		client.GetClient().Discard()
	}
}

func (fridge *Fridge) ValidateDevData(config entities.DevConfig) (bool, string) {
	var fridgeConfig FridgeConfig
	json.Unmarshal(config.Data, &fridgeConfig)

	if !entities.ValidateMAC(config.MAC) {
		log.Error("Invalid MAC")
		return false, "Invalid MAC"
	} else if !entities.ValidateCollectFreq(fridgeConfig.CollectFreq) {
		log.Error("Invalid Collect Frequency Value")
		return false, "Collect Frequency should be more than 150!"
	} else if !entities.ValidateSendFreq(fridgeConfig.SendFreq) {
		log.Error("Invalid Send Frequency Value")
		return false, "Send Frequency should be more than 150!"
	}
	return true, ""
}

func (fridge *Fridge) GetDefaultConfig() *entities.DevConfig {
	config := FridgeConfig{
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
		MAC:  fridge.Meta.MAC,
		Data: arrByte,
	}
}

func (fridge *Fridge) CheckDevConfigAndMarshal(arr []byte, configInfo, mac string, client db.Client) []byte {
	fridgeConfig := fridge.getFridgeConfig(configInfo, mac, client)
	json.Unmarshal(arr, &fridgeConfig)
	arr, _ = json.Marshal(fridgeConfig)
	return arr
}

func (fridge *Fridge) SendDefaultConfigurationTCP(conn net.Conn, dbClient db.Client, req *entities.Request) []byte {
	var config *entities.DevConfig
	configInfo := req.Meta.MAC + ":" + "config" // key
	if ok, _ := dbClient.GetClient().Exists(configInfo); ok {

		config = fridge.GetDevConfig(configInfo, req.Meta.MAC, dbClient)
		log.Println("Old Device with MAC: ", req.Meta.MAC, "detected.")

	} else {
		log.Warningln("New Device with MAC: ", req.Meta.MAC, "detected.")
		log.Warningln("Default Config will be sent.")
		config = fridge.GetDefaultConfig()

		fridge.SetDevConfig(configInfo, config, dbClient)
	}

	return config.Data
}

func (fridge *Fridge) PatchDevConfigHandlerHTTP(w http.ResponseWriter, r *http.Request, devMeta entities.DevMeta, dbClient db.Client) {
	var config *entities.DevConfig
	configInfo := devMeta.MAC + ":" + "config" // key

	err := json.NewDecoder(r.Body).Decode(&config)

	config.Data = fridge.CheckDevConfigAndMarshal(config.Data, configInfo, devMeta.MAC, dbClient)

	if err != nil {
		http.Error(w, err.Error(), 400)
		log.Errorln("NewDec: ", err)
	}

	valid, message := fridge.ValidateDevData(*config)
	if !valid {
		http.Error(w, message, 400)
	} else {
		// Save New Configuration to DB
		fridge.SetDevConfig(configInfo, config, dbClient)
		log.Println("New Config was added to DB: ", config.MAC)
		JSONConfig, _ := json.Marshal(config)
		dbClient.Publish("configChan", JSONConfig)
	}
}
