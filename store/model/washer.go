package model

import (
	"encoding/json"
	"fmt"
)

type (
	// washerData is used to store amount of turnovers and water temperature.
	washerData struct {
		Turnovers map[int64]int64   `json:"turnovers"`
		WaterTemp map[int64]float32 `json:"waterTemp"`
	}

	// washerCfg is used to store washer configuration.
	washerCfg struct {
		Name           string  `json:"name"`
		Temperature    float32 `json:"temp"`
		WashTime       int64   `json:"washTime"`
		WashTurnovers  int64   `json:"washTurnovers"`
		RinseTime      int64   `json:"rinseTime"`
		RinseTurnovers int64   `json:"rinseTurnovers"`
		SpinTime       int64   `json:"spinTime"`
		SpinTurnovers  int64   `json:"spinTurnovers"`
	}

	washer struct {
		id   DevID
		cfg  washerCfg
		data washerData
	}
)

var (
	// lightMode stores configuration for light washing mode.
	lightMode = washerCfg{
		Name:           "lightMode",
		Temperature:    60,
		WashTime:       90,
		WashTurnovers:  240,
		RinseTime:      30,
		RinseTurnovers: 120,
		SpinTime:       30,
		SpinTurnovers:  60,
	}

	// standardMode stores configuration for standard washing mode.
	standardMode = washerCfg{
		Name:           "standardMode",
		Temperature:    240,
		WashTime:       120,
		WashTurnovers:  240,
		RinseTime:      60,
		RinseTurnovers: 180,
		SpinTime:       60,
		SpinTurnovers:  60,
	}
)

func (w *washer) GetData() (*Data, error) {
	return nil, nil
}

func (w *washer) SaveData(d *Data) error {
	return nil
}

func (w *washer) GetDefaultCfg() (*Cfg, error) {
	return nil, nil
}

func (w *washer) GetCfg() (*Cfg, error) {
	return nil, nil
}

func (w *washer) SetCfg(c *Cfg) error {
	return nil
}

func (w *washer) InitCfg() error {
	return nil
}

func (w *washer) GetMeta() (*Meta, error) {
	return nil, nil
}

func (w *washer) SetMeta(m *Meta) error {
	return nil
}

func (w *washer) IsRegistered() (bool, error) {
	return false, nil
}

func parseWasher(c *Cfg, d *Data) (Devicer, error) {
	var cfg washerCfg
	if err := json.Unmarshal(c.Data, &cfg); err != nil {
		return nil, fmt.Errorf("func Unmarshal: %v", err)
	}

	var data washerData
	if err := json.Unmarshal(d.Data, &data); err != nil {
		return nil, fmt.Errorf("func Unmarshal: %v", err)
	}

	return &washer{
		cfg:  cfg,
		data: data,
	}, nil
}
