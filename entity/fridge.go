package entity

var (
	// DefaultFridgeCfg is used to store default fridge configuration.
	DefaultFridgeCfg = FridgeCfg{
		TurnedOn:    true,
		CollectFreq: 1000,
		SendFreq:    2000,
	}
)

// FridgeData is used to store temperature with timestamps for each compartment of the fridge.
type FridgeData struct {
	TopCompart map[int64]float32 `json:"topCompart"`
	BotCompart map[int64]float32 `json:"botCompart"`
}

// FridgeCfg is used to store fridge configuration.
type FridgeCfg struct {
	TurnedOn    bool  `json:"turnedOn"`
	CollectFreq int64 `json:"collectFreq"`
	SendFreq    int64 `json:"sendFreq"`
}
