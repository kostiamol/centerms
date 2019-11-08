package store

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/kostiamol/centerms/svc"

	"github.com/pkg/errors"
)

type (
	// washerData is used to store amount of turnovers and water temperature.
	washerData struct {
		Turnovers map[int64]int64   `json:"turnovers"`
		WaterTemp map[int64]float32 `json:"waterTemp"`
	}

	// washerCfg is used to store washer configuration.
	washerCfg struct {
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

	// timerMode is used to store timer settings.
	timerMode struct {
		Name      string `json:"name"`
		StartTime int64  `json:"time"`
	}
)

var (
	// lightMode stores configuration for light washing mode.
	lightMode = washerCfg{
		Name:           "lightMode",
		Temperature:    60,
		WashTime:       90,
		WashTurnovers:  240,
		RinseTime:      30,
		RinseTurnovers: 120,
		SpinTime:       30,
		SpinTurnovers:  60,
	}

	// standardMode stores configuration for standard washing mode.
	standardMode = washerCfg{
		Name:           "standardMode",
		Temperature:    240,
		WashTime:       120,
		WashTurnovers:  240,
		RinseTime:      60,
		RinseTurnovers: 180,
		SpinTime:       60,
		SpinTurnovers:  60,
	}
)

func (s *store) getWasherData(m *svc.DevMeta) (*svc.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := s.smembers(devParamsKey)
	if err != nil {
		return nil, errors.Wrap(err, "store: func getWasherData: func SMEMBERS: ")
	}

	for _, p := range params {
		data[p], err = s.zrangebyscore(devParamsKey+":"+p, "-inf", "inf")
		if err != nil {
			return nil, errors.Wrap(err, "store: func getWasherData: func ZRANGEBYSCORE: ")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "store: func getWasherData: func Marshal: ")
	}

	return &svc.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (s *store) saveWasherData(d *svc.DevData) error {
	var washer washerData
	if err := json.NewDecoder(bytes.NewBuffer(d.Data)).Decode(&washer); err != nil {
		return errors.Wrap(err, "store: func saveWasherData: func Decode: ")
	}

	devKey := partialDevKey + d.Meta.Type + ":" + d.Meta.Name + ":" + d.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := s.multi(); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: func saveWasherData: func MULTI: ")
	}
	if err := s.setTurnoversData(washer.Turnovers, paramsKey+":"+"Turnovers"); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: func saveWasherData: func setTurnoversData: ")
	}
	if err := s.setWaterTempData(washer.WaterTemp, paramsKey+":"+"WaterTemp"); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: func saveWasherData: func setWaterTempData: ")
	}
	if _, err := s.exec(); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: func saveWasherData: func EXEC: ")
	}
	return nil
}

func (s *store) setTurnoversData(tempCam map[int64]int64, key string) error {
	for t, v := range tempCam {
		_, err := s.zadd(key, strconv.FormatInt(t, 10),
			strconv.FormatInt(t, 10)+":"+strconv.FormatInt(v, 10))
		if err != nil {
			return errors.Wrap(err, "func ZADD: ")
		}
	}
	return nil
}

func (s *store) setWaterTempData(tempCam map[int64]float32, key string) error {
	for t, v := range tempCam {
		_, err := s.zadd(key, strconv.FormatInt(t, 10),
			strconv.FormatInt(t, 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
		if err != nil {
			return errors.Wrap(err, "func ZADD: ")
		}
	}
	return nil
}

func (s *store) getWasherCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	cfg, err := s.getWasherDefaultCfg(m)
	if err != nil {
		return nil, errors.Wrap(err, "store: func getWasherCfg: func getWasherDefaultCfg: ")
	}

	cfg.MAC = m.MAC
	cfgKey := m.MAC + partialDevCfgKey
	unixTime := int64(100)
	mode, err := s.zrangebyscore(cfgKey, unixTime-100, unixTime+100)
	if err != nil {
		return nil, errors.Wrap(err, "store: func getWasherCfg: func ZRANGEBYSCORE: ")
	}
	if len(mode) == 0 {
		return nil, errors.New("store: mode is empty")
	}

	l := lightMode
	cfg.Data, err = json.Marshal(l)
	if err != nil {
		return nil, errors.Wrap(err, "store: func getWasherCfg: func Marshal: ")
	}
	return cfg, nil
}

func (s *store) setWasherCfg(c *svc.DevCfg) error {
	var m *timerMode
	if err := json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&m); err != nil {
		return errors.Wrap(err, "store: func setWasherCfg: func Decode: ")
	}

	cfgKey := c.MAC + partialDevCfgKey
	if _, err := s.zadd(cfgKey, m.StartTime, m.Name); err != nil {
		return errors.Wrap(err, "store: func setWasherCfg: func ZADD: ")
	}
	return nil
}

func (s *store) getWasherDefaultCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	b, err := json.Marshal(standardMode)
	if err != nil {
		return nil, errors.Wrap(err, "func Marshal: ")
	}

	return &svc.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
