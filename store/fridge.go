package store

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/entity"
	"github.com/pkg/errors"
)

func (r *Redis) getFridgeData(m *entity.DevMeta) (*entity.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	paramsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := redis.Strings(r.conn.Do("SMEMBERS", paramsKey))
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeData(): SMembers() has failed")
	}

	for _, p := range params {
		data[p], err = redis.Strings(r.conn.Do("ZRANGEBYSCORE", paramsKey+":"+p, "-inf", "inf"))
		if err != nil {
			errors.Wrap(err, "Redis: getFridgeData(): ZRangeByScore() has failed")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeData(): fridge data marshalling has failed")
		return nil, err
	}

	return &entity.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (r *Redis) saveFridgeData(d *entity.DevData) error {
	var f entity.FridgeData
	if err := json.Unmarshal([]byte(d.Data), &f); err != nil {
		errors.Wrap(err, "Redis: saveFridgeData(): FridgeData unmarshalling has failed")
		return err
	}

	devKey := partialDevKey + d.Meta.Type + ":" + d.Meta.Name + ":" + d.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := r.conn.Do("HMSET", "devParamsKeys", paramsKey); err != nil {
		errors.Wrap(err, "Redis: saveFridgeData(): SAdd() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("HMSET", devKey, "ReqTime", d.Time); err != nil {
		errors.Wrap(err, "Redis: saveFridgeData(): HMSet() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("SADD", paramsKey, "TopCompart", "BotCompart"); err != nil {
		errors.Wrap(err, "Redis: saveFridgeData(): SAdd() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if err := r.setFridgeCompartData(f.TopCompart, paramsKey+":"+"TopCompart"); err != nil {
		errors.Wrap(err, "Redis: setFridgeCompartData(): Multi() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if err := r.setFridgeCompartData(f.BotCompart, paramsKey+":"+"BotCompart"); err != nil {
		errors.Wrap(err, "Redis: saveFridgeData(): setFridgeCompartData() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	return nil
}

func (r *Redis) setFridgeCompartData(tempCam map[int64]float32, key string) error {
	for time, temp := range tempCam {
		_, err := r.conn.Do("ZADD", key, strconv.FormatInt(int64(time), 10),
			strconv.FormatInt(int64(time), 10)+":"+strconv.FormatFloat(float64(temp), 'f', -1, 32))
		if err != nil {
			errors.Wrap(err, "Redis: setFridgeCompartData(): ZAdd() has failed")
			return err
		}
	}
	return nil
}

func (r *Redis) getFridgeCfg(m *entity.DevMeta) (*entity.DevCfg, error) {
	cfgKey := m.MAC + partialDevCfgKey

	to, err := redis.Strings(r.conn.Do("HMGET", cfgKey, "TurnedOn"))
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): TurnedOn field extraction has failed")
		return nil, err
	}
	cf, err := redis.Strings(r.conn.Do("HMGET", cfgKey, "CollectFreq"))
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): CollectFreq field extraction has failed")
		return nil, err
	}
	sf, err := redis.Strings(r.conn.Do("HMGET", cfgKey, "SendFreq"))
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): SendFreq field extraction has failed")
		return nil, err
	}

	pto, err := strconv.ParseBool(strings.Join(to, " "))
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): TurnedOn field parsing has failed")
		return nil, err
	}

	pcf, err := strconv.ParseInt(strings.Join(cf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): CollectFreq field parsing has failed")
		return nil, err
	}
	psf, err := strconv.ParseInt(strings.Join(sf, " "), 10, 64)
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): SendFreq field parsing has failed")
		return nil, err
	}

	c := entity.FridgeCfg{
		TurnedOn:    pto,
		CollectFreq: pcf,
		SendFreq:    psf,
	}

	b, err := json.Marshal(&c)
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeCfg(): FridgeCfg marshalling has failed")
		return nil, err
	}

	return &entity.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, err
}

func (r *Redis) setFridgeCfg(c *entity.DevCfg, m *entity.DevMeta) error {
	var cfg *entity.DevCfg
	if ok, err := r.DevIsRegistered(m); ok {
		if err != nil {
			errors.Wrapf(err, "Redis: setFridgeCfg(): DevIsRegistered() has failed")
			return err
		}
		if cfg, err = r.getFridgeCfg(m); err != nil {
			return err
		}
	} else {
		if err != nil {
			errors.Wrapf(err, "Redis: setFridgeCfg(): DevIsRegistered() has failed")
			return err
		}
		if cfg, err = r.getFridgeDefaultCfg(m); err != nil {
			return err
		}
	}

	var f entity.FridgeCfg
	if err := json.Unmarshal(cfg.Data, &f); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): Unmarshal() has failed")
		return err
	}

	if err := json.Unmarshal(c.Data, &f); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): Unmarshal() has failed")
		return err
	}

	cfgKey := c.MAC + partialDevCfgKey
	if _, err := r.conn.Do("MULTI"); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): Multi() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("HMSET", cfgKey, "TurnedOn", f.TurnedOn); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("HMSET", cfgKey, "CollectFreq", f.CollectFreq); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("HMSET", cfgKey, "SendFreq", f.SendFreq); err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): HMSet() has failed")
		r.conn.Do("DISCARD")
		return err
	}

	_, err := r.conn.Do("EXEC")
	if err != nil {
		errors.Wrap(err, "Redis: setFridgeCfg(): Exec() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	return nil
}

func (r *Redis) getFridgeDefaultCfg(m *entity.DevMeta) (*entity.DevCfg, error) {
	b, err := json.Marshal(entity.DefaultFridgeCfg)
	if err != nil {
		errors.Wrap(err, "Redis: getFridgeDefaultCfg(): FridgeCfg marshalling has failed")
		return nil, err
	}

	return &entity.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
