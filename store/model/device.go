package model

import (
	"encoding/json"
	"fmt"
)

type (
	//Devicer represents an interface to an abstract device.
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

	Type string

	// Meta .
	Meta struct {
		Type Type   `json:"type"`
		Name string `json:"name"`
		MAC  string `json:"mac"`
	}

	// Data .
	Data struct {
		Time int64           `json:"time"`
		Meta Meta            `json:"meta"`
		Data json.RawMessage `json:"data"`
	}

	// Cfg .
	Cfg struct {
		MAC  string          `json:"mac"`
		Data json.RawMessage `json:"data"`
	}
)

const (
	Fridge Type = "fridge"
	Washer Type = "washer"
)

func NewDevice(id string, t Type) (Devicer, error) {
	switch t {
	case Fridge:
		return &fridge{id: id}, nil
	case Washer:
		return &washer{id: id}, nil
	default:
		return nil, fmt.Errorf("unknown device with type: %s", t)
	}
}
