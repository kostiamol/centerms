package model

import (
	"encoding/json"
	"fmt"
)

type (
	Devicer interface {
		GetData() (*Data, error)
		SaveData(*Data) error

		GetCfg() (*Cfg, error)
		GetDefaultCfg() (*Cfg, error)
		SetCfg(*Cfg) error

		GetMeta() (*Meta, error)
		SetMeta(*Meta) error

		IsRegistered() (bool, error)
	}

	DevID string
	Type  string

	// Meta is used to subscriber device metadata: it's type, name (model) and MAC address.
	Meta struct {
		Type Type   `json:"type"`
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
	Fridge Type = "fridge"
	Washer Type = "washer"
)

func DefineDevice(t Type, id DevID) (Devicer, error) {
	switch t {
	case Fridge:
		return &fridge{id: id}, nil
	case Washer:
		return &washer{id: id}, nil
	default:
		return nil, fmt.Errorf("unknown device with type: %s", t)
	}
}
