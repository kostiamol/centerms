package entities

type Fridge struct {
	Data   FridgeData   `json:"data"`
	Config FridgeConfig `json:"config"`
	Meta   DevMeta      `json:"meta"`
}

type FridgeData struct {
	TempCam1 map[int64]float32 `json:"tempCam1"`
	TempCam2 map[int64]float32 `json:"tempCam2"`
}

type FridgeConfig struct {
	TurnedOn    bool  `json:"turnedOn"`
	StreamOn    bool  `json:"streamOn"`
	CollectFreq int64 `json:"collectFreq"`
	SendFreq    int64 `json:"sendFreq"`
}
