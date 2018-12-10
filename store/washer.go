package store

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/entity"
	"github.com/pkg/errors"
)

func (r *Redis) getWasherData(m *entity.DevMeta) (*entity.DevData, error) {
	devKey := partialDevKey + m.Type + ":" + m.Name + ":" + m.MAC
	devParamsKey := devKey + partialDevParamsKey

	data := make(map[string][]string)
	params, err := redis.Strings(r.conn.Do("SMEMBERS", devParamsKey))
	if err != nil {
		errors.Wrap(err, "RedisDevStore: getWasherData(): can't read members from devParamsKeys")
	}

	for _, p := range params {
		data[p], err = redis.Strings(r.conn.Do("ZRANGEBYSCORE", devParamsKey+":"+p, "-inf", "inf"))
		if err != nil {
			errors.Wrap(err, "RedisDevStore: getWasherData(): can't read members from sorted set")
		}
	}

	b, err := json.Marshal(&data)
	if err != nil {
		errors.Wrap(err, "Redis: getWasherData(): washer data marshalling has failed")
		return nil, err
	}

	return &entity.DevData{
		Meta: *m,
		Data: b,
	}, err
}

func (r *Redis) saveWasherData(d *entity.DevData) error {
	var w entity.WasherData
	if err := json.NewDecoder(bytes.NewBuffer(d.Data)).Decode(&w); err != nil {
		errors.Wrap(err, "RedisDevStore: saveWasherData(): WasherData decoding has failed")
		return err
	}

	devKey := partialDevKey + d.Meta.Type + ":" + d.Meta.Name + ":" + d.Meta.MAC
	paramsKey := devKey + partialDevParamsKey

	if _, err := r.conn.Do("MULTI"); err != nil {
		errors.Wrap(err, "RedisDevStore: saveWasherData(): Multi() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if err := r.setTurnoversData(w.Turnovers, paramsKey+":"+"Turnovers"); err != nil {
		errors.Wrap(err, "RedisDevStore: saveWasherData(): setTurnoversData() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if err := r.setWaterTempData(w.WaterTemp, paramsKey+":"+"WaterTemp"); err != nil {
		errors.Wrap(err, "RedisDevStore: saveWasherData(): setWaterTempData() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	if _, err := r.conn.Do("EXEC"); err != nil {
		errors.Wrap(err, "RedisDevStore: saveWasherData(): Exec() has failed")
		r.conn.Do("DISCARD")
		return err
	}
	return nil
}

func (r *Redis) setTurnoversData(tempCam map[int64]int64, key string) error {
	for t, v := range tempCam {
		_, err := r.conn.Do("ZADD", key, strconv.FormatInt(int64(t), 10),
			strconv.FormatInt(int64(t), 10)+":"+strconv.FormatInt(int64(v), 10))
		if err != nil {
			errors.Wrap(err, "RedisDevStore: setTurnoversData(): adding to sorted set has failed")
			return err
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
			errors.Wrap(err, "RedisDevStore: setWaterTempData(): adding to sorted set has failed")
			return err
		}
	}
	return nil
}

func (r *Redis) getWasherCfg(m *entity.DevMeta) (*entity.DevCfg, error) {
	c, err := r.getWasherDefaultCfg(m)
	if err != nil {
		errors.Wrap(err, "RedisDevStore: getWasherCfg(): getWasherDefaultCfg() has failed")
	}

	c.MAC = m.MAC
	cfgKey := m.MAC + partialDevCfgKey
	unixTime := int64(100) // fake
	mode, err := redis.Strings(r.conn.Do("ZRANGEBYSCORE", cfgKey, unixTime-100, unixTime+100))
	if err != nil {
		errors.Wrap(err, "RedisDevStore: getWasherCfg(): ZRangeByScore() has failed")
	}
	if len(mode) == 0 {
		return c, err
	}

	l := entity.LightMode
	c.Data, err = json.Marshal(l)
	if err != nil {
		errors.Wrap(err, "RedisDevStore: getWasherCfg(): WasherCfg marshalling has failed")
	}
	return c, err
}

func (r *Redis) setWasherCfg(c *entity.DevCfg) error {
	var m *entity.TimerMode
	err := json.NewDecoder(bytes.NewBuffer(c.Data)).Decode(&m)
	if err != nil {
		errors.Wrap(err, "RedisDevStorage: setWasherCfg(): Decode() has failed")
	}

	cfgKey := c.MAC + partialDevCfgKey
	_, err = r.conn.Do("ZADD", cfgKey, m.StartTime, m.Name)
	if err != nil {
		errors.Wrap(err, "RedisDevStore: setWasherCfg(): ZAdd() has failed")
	}
	return err
}

func (r *Redis) getWasherDefaultCfg(m *entity.DevMeta) (*entity.DevCfg, error) {
	b, err := json.Marshal(entity.StandardMode)
	if err != nil {
		errors.Wrap(err, "RedisDevStore: getWasherDefaultCfg(): WasherConf marshalling has failed")
		return nil, err
	}

	return &entity.DevCfg{
		MAC:  m.MAC,
		Data: b,
	}, nil
}
