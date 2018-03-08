package entities

var (
	// LightMode stores config for light washing mode.
	LightMode = WasherConfig{
		Name:           "LightMode",
		Temperature:    60,
		WashTime:       90,
		WashTurnovers:  240,
		RinseTime:      30,
		RinseTurnovers: 120,
		SpinTime:       30,
		SpinTurnovers:  60,
	}

	// FastMode stores config for fast washing mode.
	FastMode = WasherConfig{
		Name:           "FastMode",
		Temperature:    180,
		WashTime:       30,
		WashTurnovers:  300,
		RinseTime:      15,
		RinseTurnovers: 240,
		SpinTime:       15,
		SpinTurnovers:  60,
	}

	// StandardMode stores config for standard washing mode.
	StandardMode = WasherConfig{
		Name:           "StandardMode",
		Temperature:    240,
		WashTime:       120,
		WashTurnovers:  240,
		RinseTime:      60,
		RinseTurnovers: 180,
		SpinTime:       60,
		SpinTurnovers:  60,
	}
)

// WasherData is used to store amount of turnovers and water temperature.
type WasherData struct {
	Turnovers map[int64]int64   `json:"turnovers"`
	WaterTemp map[int64]float32 `json:"waterTemp"`
}

// WasherConfig is used to store washer config.
type WasherConfig struct {
	Name           string  `json:"name"`
	MAC            string  `json:"mac"`
	Temperature    float32 `json:"temp"`
	WashTime       int64   `json:"washTime"`
	WashTurnovers  int64   `json:"washTurnovers"`
	RinseTime      int64   `json:"rinseTime"`
	RinseTurnovers int64   `json:"rinseTurnovers"`
	SpinTime       int64   `json:"spinTime"`
	SpinTurnovers  int64   `json:"spinTurnovers"`
}

// TimerMode is used to store timer settings.
type TimerMode struct {
	Name      string `json:"name"`
	StartTime int64  `json:"time"`
}
