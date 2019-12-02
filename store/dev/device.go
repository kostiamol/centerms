package dev

import (
	"encoding/json"
	"fmt"
)

type (
	Devicer interface {
		GetData() (*Data, error)
		SaveData(*Data) error

		GetCfg(id string) (*Cfg, error)
		SetCfg(*Cfg) error

		GetMeta() (*Meta, error)
		SetMeta(*Meta) error

		IsRegistered() (bool, error)
	}

	// Meta is used to subscriber device metadata: it's type, name (model) and MAC address.
	Meta struct {
		Type string `json:"type"`
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

	Type string
)

const (
	Fridge Type = "fridge"
	Washer Type = "washer"
)

func defineDevice(t Type) (Devicer, error) {
	switch t {
	case Fridge:
		return &fridge{}, nil
	case Washer:
		return &washer{}, nil
	default:
		return nil, fmt.Errorf("unknown device with type: %s", t)
	}
}
