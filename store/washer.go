package store

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/kostiamol/centerms/svc"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
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

	// fastMode stores configuration for fast washing mode.
	fastMode = washerCfg{
		Name:           "fastMode",
		Temperature:    180,
		WashTime:       30,
		WashTurnovers:  300,
		RinseTime:      15,
		RinseTurnovers: 240,
		SpinTime:       15,
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

// washerData is used to store amount of turnovers and water temperature.
type washerData struct {
	Turnovers map[int64]int64   `json:"turnovers"`
	WaterTemp map[int64]float32 `json:"waterTemp"`
}

// washerCfg is used to store washer configuration.
type washerCfg struct {
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
type timerMode struct {
	Name      string `json:"name"`
	StartTime int64  `json:"time"`
}

func (r *Redis) getWasherData(m *svc.DevMeta) (*svc.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := redis.Strings(r.conn.Do("SMEMBERS", devParamsKey))
	if err != nil {
		errors.Wrap(err, "store: getWasherData(): SMEMBERS() failed: ")
	}

	for _, p := range params {
		data[p], err = redis.Strings(r.conn.Do("ZRANGEBYSCORE", devParamsKey+":"+p, "-inf", "inf"))
		if err != nil {
			errors.Wrap(err, "store: getWasherData(): ZRANGEBYSCORE() failed: ")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		errors.Wrap(err, "store: getWasherData(): Marshal() failed: ")
		return nil, err
	}

	return &svc.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (r *Redis) saveWasherData(d *svc.DevData) error {
	var washer washerData
	if err := json.NewDecoder(bytes.NewBuffer(d.Data)).Decode(&washer); err != nil {
		return errors.Wrap(err, "store: saveWasherData(): Decode() failed: ")
	}

	devKey := partialDevKey + d.Meta.Type + ":" + d.Meta.Name + ":" + d.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := r.conn.Do("MULTI"); err != nil {
		r.conn.Do("DISCARD")
		return errors.Wrap(err, "store: saveWasherData(): Multi() failed: ")
	}
	if err := r.setTurnoversData(washer.Turnovers, paramsKey+":"+"Turnovers"); err != nil {
		r.conn.Do("DISCARD")
		return errors.Wrap(err, "store: saveWasherData(): setTurnoversData() failed: ")
	}
	if err := r.setWaterTempData(washer.WaterTemp, paramsKey+":"+"WaterTemp"); err != nil {
		r.conn.Do("DISCARD")
		return errors.Wrap(err, "store: saveWasherData(): setWaterTempData() failed: ")
	}
	if _, err := r.conn.Do("EXEC"); err != nil {
		r.conn.Do("DISCARD")
		return errors.Wrap(err, "store: saveWasherData(): EXEC() failed: ")
	}
	return nil
}

func (r *Redis) setTurnoversData(tempCam map[int64]int64, key string) error {
	for t, v := range tempCam {
		_, err := r.conn.Do("ZADD", key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
		if err != nil {
			return errors.Wrap(err, "ZADD() failed: ")
		}
	}
	return nil
}

func (r *Redis) setWaterTempData(tempCam map[int64]float32, key string) error {
	for t, v := range tempCam {
		_, err := r.conn.Do("ZADD", key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+
				strconv.FormatFloat(float64(v), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "ZADD() failed: ")
			return err
		}
	}
	return nil
}

func (r *Redis) getWasherCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	cfg, err := r.getWasherDefaultCfg(m)
	if err != nil {
		errors.Wrap(err, "store: getWasherCfg(): getWasherDefaultCfg(): ")
	}

	cfg.MAC = m.MAC
	cfgKey := m.MAC + partialDevCfgKey
	unixTime := int64(100) // fake
	mode, err := redis.Strings(r.conn.Do("ZRANGEBYSCORE", cfgKey, unixTime-100, unixTime+100))
	if err != nil {
		return nil, errors.Wrap(err, "store: getWasherCfg(): ZRANGEBYSCORE() failed: ")
	}
	if len(mode) == 0 {
		return nil, errors.New("store: mode is empty")
	}

	l := lightMode
	cfg.Data, err = json.Marshal(l)
	if err != nil {
		return nil, errors.Wrap(err, "store: getWasherCfg(): Marshal() failed: ")
	}
	return cfg, nil
}

func (r *Redis) setWasherCfg(c *svc.DevCfg) error {
	var m *timerMode
	if err := json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&m); err != nil {
		return errors.Wrap(err, "store: setWasherCfg(): Decode() failed: ")
	}

	cfgKey := c.MAC + partialDevCfgKey
	if _, err := r.conn.Do("ZADD", cfgKey, m.StartTime, m.Name); err != nil {
		return errors.Wrap(err, "store: setWasherCfg(): ZADD() failed: ")
	}
	return nil
}

func (r *Redis) getWasherDefaultCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	b, err := json.Marshal(standardMode)
	if err != nil {
		return nil, errors.Wrap(err, "Marshal() failed: ")
	}

	return &svc.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
