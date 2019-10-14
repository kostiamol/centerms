package store

import (
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/svc"
	"github.com/pkg/errors"
)

const (
	fridge = "fridge"
	washer = "washer"
)

// GetDevsData returns the data concerning all the devices.
func (s *store) GetDevsData() ([]svc.DevData, error) {
	devParamsKeys, err := s.smembers("devParamsKeys")
	if err != nil {
		return nil, errors.Wrap(err, "store: GetDevsData(): SMEMBERS() for devParamsKeys failed: ")
	}

	devParamsKeysTokens := make([][]string, len(devParamsKeys))
	for k, v := range devParamsKeys {
		devParamsKeysTokens[k] = strings.Split(v, ":")
	}

	var (
		devData  svc.DevData
		devsData []svc.DevData
	)

	for index, key := range devParamsKeysTokens {
		devData.Meta = svc.DevMeta{
			Type: key[1],
			Name: key[2],
			MAC:  key[3],
		}
		data := make(map[string][]string)
		params, err := s.smembers(devParamsKeys[index])
		if err != nil {
			return nil, errors.Wrapf(err, "store: GetDevsData(): SMEMBERS() for %s failed: ", devParamsKeys[index])
		}

		for i, p := range params {
			data[p], err = s.zrangebyscore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				return nil, errors.Wrapf(err, "store: GetDevsData(): ZRANGEBYSCORE() for %s failed: ", devParamsKeys[i])
			}
		}
		devsData = append(devsData, devData)
	}
	return devsData, err
}

// GetDevMeta returns metadata for the given device.
func (s *store) GetDevMeta(id string) (*svc.DevMeta, error) {
	t, err := s.hget("id:"+id, "type")
	if err != nil {
		return nil, errors.Wrap(err, "store: GetDevMeta(): HGET() for type failed: ")
	}
	n, err := s.hget("id:"+id, "name")
	if err != nil {
		return nil, errors.Wrap(err, "store: GetDevMeta():  HGET() for name failed: ")
	}
	m, err := s.hget("id:"+id, "mac")
	if err != nil {
		return nil, errors.Wrap(err, "store: GetDevMeta():  HGET() for mac failed: ")
	}

	return &svc.DevMeta{
		Type: t,
		Name: n,
		MAC:  m,
	}, nil
}

// SetDevMeta sets metadata for the given device.
func (s *store) SetDevMeta(m *svc.DevMeta) error {
	if _, err := s.hmset("id:"+m.MAC, "type", m.Type); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSET() for type failed: ")
	}
	if _, err := s.hmset("id:"+m.MAC, "name", m.Name); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSET() for name failed: ")
	}
	if _, err := s.hmset("id:"+m.MAC, "mac", m.MAC); err != nil {
		return errors.Wrap(err, "store: setFridgeCfg(): HMSET() for mac failed: ")
	}
	return nil
}

// DevIsRegistered returns 'true' if the given device is registered, otherwise - 'false'.
func (s *store) DevIsRegistered(m *svc.DevMeta) (bool, error) {
	cfgKey := m.MAC + partialDevCfgKey
	if ok, err := s.exists(cfgKey); ok {
		if err != nil {
			return false, errors.Wrap(err, "store: DevIsRegistered(): EXISTS() failed: ")
		}
		return true, nil
	}
	return false, nil
}

// GetDevData returns data concerning given device.
func (s *store) GetDevData(id string) (*svc.DevData, error) {
	m, err := s.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch m.Type {
	case fridge:
		return s.getFridgeData(m)
	case washer:
		return s.getWasherData(m)
	default:
		return nil, errors.New("store: GetDevData(): dev type is unknown")
	}
}

// SaveDevData saves data concerning given device.
func (s *store) SaveDevData(d *svc.DevData) error {
	switch d.Meta.Type {
	case fridge:
		return s.saveFridgeData(d)
	case washer:
		return s.saveWasherData(d)
	default:
		return errors.New("store: SaveDevData(): dev type is unknown")
	}
}

// GetDevCfg returns configuration for the given device.
func (s *store) GetDevCfg(id string) (*svc.DevCfg, error) {
	m, err := s.GetDevMeta(id)
	if err != nil {
		return nil, err
	}

	switch m.Type {
	case fridge:
		return s.getFridgeCfg(m)
	case washer:
		return s.getWasherCfg(m)
	default:
		return nil, errors.New("store: GetDevCfg(): dev type is unknown")
	}
}

// SetDevCfg sets configuration for the given device.
func (s *store) SetDevCfg(id string, c *svc.DevCfg) error {
	m, err := s.GetDevMeta(id)
	if err != nil {
		return err
	}

	switch m.Type {
	case fridge:
		return s.setFridgeCfg(c, m)
	case washer:
		return s.setWasherCfg(c)
	default:
		return errors.New("store: SetDevCfg(): dev type is unknown")
	}
}

// GetDevDefaultCfg returns default configuration for the given device.
func (s *store) GetDevDefaultCfg(m *svc.DevMeta) (*svc.DevCfg, error) {
	switch m.Type {
	case fridge:
		return s.getFridgeDefaultCfg(m)
	case washer:
		return s.getWasherDefaultCfg(m)
	default:
		return nil, errors.New("store: GetDevDefaultCfg(): dev type is unknown")
	}
}

// Publish posts a message on the given channel.
func (s *store) Publish(msg interface{}, channel string) (int64, error) {
	numberOfClients, err := redis.Int64(s.publish(msg, channel))
	if err != nil {
		return 0, errors.Wrap(err, "store: PUBLISH() failed: ")
	}
	return numberOfClients, nil
}

// Subscribe subscribes the client to the specified channels.
func (s *store) Subscribe(c chan []byte, channel ...string) error {
	conn := redis.PubSubConn{Conn: s.pool.Get()}
	for _, cn := range channel {
		if err := conn.Subscribe(cn); err != nil {
			return errors.Wrap(err, "store: SUBSCRIBE() failed: ")
		}
	}
	for {
		switch v := conn.Receive().(type) {
		case redis.Message:
			c <- v.Data
		}
	}
}