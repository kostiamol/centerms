package model

import (
	"encoding/json"
	"fmt"
)

type (
	// fridgeData is used to store temperature with timestamps for each compartment of the fridge.
	fridgeData struct {
		TopCompart map[int64]float32 `json:"topCompart"`
		BotCompart map[int64]float32 `json:"botCompart"`
	}

	// fridgeCfg is used to store fridge configuration.
	fridgeCfg struct {
		TurnedOn    bool  `json:"turnedOn"`
		CollectFreq int64 `json:"collectFreq"`
		SendFreq    int64 `json:"sendFreq"`
	}

	fridge struct {
		devID string
		cfg   fridgeCfg
		data  fridgeData
	}
)

// defaultFridgeCfg is used to store default fridge configuration.
var defaultFridgeCfg = fridgeCfg{
	TurnedOn:    true,
	CollectFreq: 1000,
	SendFreq:    2000,
}

func (f *fridge) GetData() (*Data, error) {
	return nil, nil
}

func (f *fridge) SaveData(d *Data) error {
	return nil
}

func (f *fridge) GetDefaultCfg() (*Cfg, error) {
	return nil, nil
}

func (f *fridge) GetCfg() (*Cfg, error) {
	return nil, nil
}

func (f *fridge) SetCfg(c *Cfg) error {
	return nil
}

func (f *fridge) SetInitCfg() error {
	return nil
}

func (f *fridge) GetMeta() (*Meta, error) {
	return nil, nil
}

func (f *fridge) SetMeta(*Meta) error {
	return nil
}

func (f *fridge) IsRegistered() (bool, error) {
	return false, nil
}

func parseFridge(c *Cfg, d *Data) (Devicer, error) {
	var cfg fridgeCfg
	if err := json.Unmarshal(c.Data, &cfg); err != nil {
		return nil, fmt.Errorf("func Unmarshal: %v", err)
	}

	var data fridgeData
	if err := json.Unmarshal(d.Data, &data); err != nil {
		return nil, fmt.Errorf("func Unmarshal: %v", err)
	}

	return &fridge{
		cfg:  cfg,
		data: data,
	}, nil
}
