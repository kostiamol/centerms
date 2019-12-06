package model

import (
	"encoding/json"
	"fmt"
)

type (
	devicer interface {
		GetData() (*Data, error)
		SaveData(*Data) error

		GetCfg() (*Cfg, error)
		GetDefaultCfg() (*Cfg, error)
		SetCfg(*Cfg) error

		GetMeta() (*Meta, error)
		SetMeta(*Meta) error

		IsRegistered() (bool, error)
	}

	UserID string
	DevID string
	DevType string

	// Meta is used to subscriber device metadata: it's type, name (model) and MAC address.
	Meta struct {
		Type DevType `json:"type"`
		Name string `json:"name"`
		MAC  string `json:"mac"`
	}

	// Data is used to subscriber time of the request, device's metadata and the data it transfers.
	Data struct {
		Time int64           `json:"time"`
		Meta Meta            `json:"meta"`
		Data json.RawMessage `json:"data"`
	}

	// Cfg holds device's MAC address and config.
	Cfg struct {
		MAC  string          `json:"mac"`
		Data json.RawMessage `json:"data"`
	}
)

const (
	Fridge DevType = "fridge"
	Washer DevType = "washer"
)

func DefineDevice(t DevType) (devicer, error) {
	switch t {
	case Fridge:
		return &fridge{}, nil
	case Washer:
		return &washer{}, nil
	default:
		return nil, fmt.Errorf("unknown device with type: %s", t)
	}
}
