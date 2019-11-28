package dev

import (
	"encoding/json"
)

type (
	Device interface {
		GetData(*Meta) error
		SaveData(*Data) error
		GetCfg() error
		GetDefaultCfg() error
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
)
