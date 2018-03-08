package entities

var (
	// DefaultFridgeConfig is used to store default fridge config.
	DefaultFridgeConfig = FridgeConfig{
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

// FridgeConfig is used to store fridge config.
type FridgeConfig struct {
	TurnedOn    bool  `json:"turnedOn"`
	CollectFreq int64 `json:"collectFreq"`
	SendFreq    int64 `json:"sendFreq"`
}
