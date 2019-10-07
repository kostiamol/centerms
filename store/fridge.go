package store

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kostiamol/centerms/svc"

	"github.com/pkg/errors"
)

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
)

// defaultFridgeCfg is used to store default fridge configuration.
var defaultFridgeCfg = fridgeCfg{
	TurnedOn:    true,
	CollectFreq: 1000,
	SendFreq:    2000,
}

func (s *store) getFridgeData(m *svc.DevMeta) (*svc.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	paramsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := s.smembers(paramsKey)
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeData(): SMEMBERS() failed: ")
	}

	for _, p := range params {
		data[p], err = s.zrangebyscore(paramsKey+":"+p, "-inf", "inf")
		if err != nil {
			return nil, errors.Wrap(err, "store: getFridgeData(): ZRANGEBYSCORE() failed: ")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeData(): Marshal() failed: ")
	}

	return &svc.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (s *store) saveFridgeData(d *svc.DevData) error {
	var fridge fridgeData
	if err := json.Unmarshal([]byte(d.Data), &fridge); err != nil {
		return errors.Wrap(err, "store: saveFridgeData(): Unmarshal() failed: ")
	}

	devKey := partialDevKey + d.Meta.Type + ":" + d.Meta.Name + ":" + d.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := s.hmset("devParamsKeys", paramsKey); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: saveFridgeData(): SADD() failed: ")
	}
	if _, err := s.hmset(devKey, "ReqTime", fmt.Sprint(d.Time)); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: saveFridgeData(): HMSET() failed: ")
	}
	if _, err := s.sadd(paramsKey, "TopCompart", "BotCompart"); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: saveFridgeData(): SADD() failed: ")
	}
	if err := s.saveFridgeCompartData(fridge.TopCompart, paramsKey+":"+"TopCompart"); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: saveFridgeData(): saveFridgeCompartData(): ")
	}
	if err := s.saveFridgeCompartData(fridge.BotCompart, paramsKey+":"+"BotCompart"); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: saveFridgeData(): saveFridgeCompartData(): ")
	}
	return nil
}

func (s *store) saveFridgeCompartData(tempCam map[int64]float32, key string) error {
	for time, temp := range tempCam {
		_, err := s.zadd(key, strconv.FormatInt(time, 10),
			strconv.FormatInt(time, 10)+":"+
				strconv.FormatFloat(float64(temp), 'f', -1, 32))
		if err != nil {
			return errors.Wrap(err, "ZADD() failed: ")
		}
	}
	return nil
}

func (s *store) getFridgeCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	cfgKey := m.MAC + partialDevCfgKey

	to, err := s.hmget("HMGET", cfgKey, "TurnedOn")
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): HMGET() for TurnedOn failed: ")
	}
	cf, err := s.hmget(cfgKey, "CollectFreq")
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): HMGET() for CollectFreq failed: ")
	}
	sf, err := s.hmget(cfgKey, "SendFreq")
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): HMGET() for SendFreq failed: ")
	}

	pto, err := strconv.ParseBool(strings.Join(to, " "))
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): ParseBool() for TurnedOn failed: ")
	}
	pcf, err := strconv.ParseInt(strings.Join(cf, " "), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): ParseInt() for CollectFreq failed: ")
	}
	psf, err := strconv.ParseInt(strings.Join(sf, " "), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): ParseInt() for SendFreq failed: ")
	}

	c := fridgeCfg{
		TurnedOn:    pto,
		CollectFreq: pcf,
		SendFreq:    psf,
	}

	b, err := json.Marshal(&c)
	if err != nil {
		return nil, errors.Wrap(err, "store: getFridgeCfg(): Marshal() failed: ")
	}

	return &svc.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, err
}

func (s *store) setFridgeCfg(c *svc.DevCfg, m *svc.DevMeta) error {
	var cfg *svc.DevCfg
	if ok, err := s.DevIsRegistered(m); ok {
		if err != nil {
			return errors.Wrapf(err, "store: setFridgeCfg(): ")
		}
		if cfg, err = s.getFridgeCfg(m); err != nil {
			return err
		}
	} else {
		if err != nil {
			return errors.Wrapf(err, "store: setFridgeCfg(): ")
		}
		if cfg, err = s.getFridgeDefaultCfg(m); err != nil {
			return errors.Wrap(err, "store: setFridgeCfg(): getFridgeDefaultCfg(): ")
		}
	}

	var fridge fridgeCfg
	if err := json.Unmarshal(cfg.Data, &fridge); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): Unmarshal() failed: ")
	}

	if err := json.Unmarshal(c.Data, &fridge); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): Unmarshal() failed: ")
	}

	cfgKey := c.MAC + partialDevCfgKey
	if _, err := s.multi(); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: setFridgeCfg(): MULTI() failed: ")
	}
	if _, err := s.hmset(cfgKey, "TurnedOn", fmt.Sprint(fridge.TurnedOn)); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: setFridgeCfg(): HMSET() failed: ")
	}
	if _, err := s.hmset(cfgKey, "CollectFreq", fmt.Sprint(fridge.CollectFreq)); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: setFridgeCfg(): HMSET() failed: ")
	}
	if _, err := s.hmset(cfgKey, "SendFreq", fmt.Sprint(fridge.SendFreq)); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: setFridgeCfg(): HMSet() failed: ")
	}

	if _, err := s.exec(); err != nil {
		_, err = s.discard()
		return errors.Wrap(err, "store: setFridgeCfg(): EXEC() failed: ")
	}
	return nil
}

func (s *store) getFridgeDefaultCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	b, err := json.Marshal(defaultFridgeCfg)
	if err != nil {
		return nil, errors.Wrap(err, "Marshal() failed: ")
	}

	return &svc.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
