package entities

var (
	DefaultFridgeConfig = FridgeConfig{
		TurnedOn:    true,
		StreamOn:    false,
		CollectFreq: 1000,
		SendFreq:    2000,
	}
)

type Fridge struct {
	Data   FridgeData   `json:"data"`
	Config FridgeConfig `json:"config"`
	Meta   DevMeta      `json:"meta"`
}

type FridgeData struct {
	TempTopCompart map[int64]float32 `json:"tempTopCompart"`
	TempBotCompart map[int64]float32 `json:"tempBotCompart"`
}

type FridgeConfig struct {
	TurnedOn    bool  `json:"turnedOn"`
	StreamOn    bool  `json:"streamOn"`
	CollectFreq int64 `json:"collectFreq"`
	SendFreq    int64 `json:"sendFreq"`
}
