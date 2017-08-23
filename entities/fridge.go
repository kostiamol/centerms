package entities

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type Fridge struct {
	Data   FridgeData   `json:"data"`
	Config FridgeConfig `json:"config"`
	Meta   DevMeta      `json:"meta"`
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

func ValidateFridge(config DevConfig) (bool, error) {
	var fridgeConfig FridgeConfig
	json.Unmarshal(config.Data, &fridgeConfig)

	if !ValidateMAC(config.MAC) {
		return false, errors.New("Invalid MAC")
	} else if !ValidateCollectFreq(fridgeConfig.CollectFreq) {
		return false, errors.New("Collect Frequency should be more than 150")
	} else if !ValidateSendFreq(fridgeConfig.SendFreq) {
		return false, errors.New("Send Frequency should be more than 150")
	}

	return true, nil
}
