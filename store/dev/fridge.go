package dev

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
		cfg  fridgeCfg
		data fridgeData
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

func (f *fridge) GetCfg() (*Cfg, error) {
	return nil, nil
}

func (f *fridge) GetDefaultCfg() (*Cfg, error) {
	return nil, nil
}

func (f *fridge) SetCfg(c *Cfg) error {
	return nil
}

func (f *fridge) GetMeta() (*Meta, error) {
	return nil, nil
}

func (f *fridge) SetDevMeta(*Meta) error {
	return nil
}

func (f *fridge) IsRegistered() (bool, error) {
	return false, nil
}
