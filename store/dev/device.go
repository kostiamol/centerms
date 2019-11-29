package dev

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
		SetDevMeta(*Meta) error
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

func New(c *Cfg, d *Data) (Devicer, error) {
	switch d.Meta.Type {
	case Fridge:
		return parseFridge(c, d)
	case Washer:
		return parseWasher(c, d)
	default:
		return nil, fmt.Errorf("unknown device with type: %s", d.Meta.Type)
	}
}
