package store

import (
	"fmt"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/kostiamol/centerms/svc"
)

const (
	fridge = "fridge"
	washer = "washer"
)

// GetDevsData returns the data concerning all the devices.
func (s *store) GetDevsData() ([]svc.DevData, error) {
	devParamsKeys, err := s.smembers("devParamsKeys")
	if err != nil {
		return nil, fmt.Errorf("store: func GetDevsData: func SMEMBERS: %v", err)
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
			return nil, fmt.Errorf("store: func GetDevsData: func SMEMBERS: %v", err)
		}

		for _, p := range params {
			data[p], err = s.zrangebyscore(devParamsKeys[index]+":"+p, "-inf", "inf")
			if err != nil {
				return nil, fmt.Errorf("store: func GetDevsData: func ZRANGEBYSCORE: %v", err)
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
		return nil, fmt.Errorf("store: func GetDevMeta: func HGET type: %v", err)
	}
	n, err := s.hget("id:"+id, "name")
	if err != nil {
		return nil, fmt.Errorf("store: func GetDevMeta: func HGET name: %v", err)
	}
	m, err := s.hget("id:"+id, "mac")
	if err != nil {
		return nil, fmt.Errorf("store: func GetDevMeta: func HGET mac: %v", err)
	}

	return &svc.DevMeta{
		Type: t,
		Name: n,
		MAC:  m,
	}, nil
}

// SetDevMeta sets metadata for the given device.
func (s *store) SetDevMeta(m *svc.DevMeta) error {
	if _, err := s.hmset("id:"+m.Type, "type", m.Type); err != nil {
		return fmt.Errorf("store: func setFridgeCfg: func HMSET type: %v", err)
	}
	if _, err := s.hmset("id:"+m.Name, "name", m.Name); err != nil {
		return fmt.Errorf("store: func setFridgeCfg: func HMSET name: %v", err)
	}
	if _, err := s.hmset("id:"+m.MAC, "mac", m.MAC); err != nil {
		return fmt.Errorf("store: func setFridgeCfg: func HMSET mac: %v", err)
	}
	return nil
}

// DevIsRegistered returns 'true' if the given device is registered, otherwise - 'false'.
func (s *store) DevIsRegistered(m *svc.DevMeta) (bool, error) {
	cfgKey := m.MAC + partialDevCfgKey
	if ok, err := s.exists(cfgKey); ok {
		if err != nil {
			return false, fmt.Errorf("store: func DevIsRegistered: func EXISTS: %v", err)
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
		return nil, fmt.Errorf("store: func GetDevData: dev type is unknown")
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
		return fmt.Errorf("store: func SaveDevData: dev type is unknown")
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
		return nil, fmt.Errorf("store: func GetDevCfg: dev type is unknown")
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
		return fmt.Errorf("store: func SetDevCfg: dev type is unknown")
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
		return nil, fmt.Errorf("store: func GetDevDefaultCfg: dev type is unknown")
	}
}

// Publish posts a message on the given channel.
func (s *store) Publish(msg interface{}, channel string) (int64, error) {
	numberOfClients, err := redis.Int64(s.publish(msg, channel))
	if err != nil {
		return 0, fmt.Errorf("store: func PUBLISH: %v", err)
	}
	return numberOfClients, nil
}

// Subscribe subscribes the client to the specified channels.
func (s *store) Subscribe(c chan []byte, channel ...string) error {
	conn := redis.PubSubConn{Conn: s.pool.Get()}
	for _, cn := range channel {
		if err := conn.Subscribe(cn); err != nil {
			return fmt.Errorf("store: func SUBSCRIBE: %v", err)
		}
	}
	for {
		switch v := conn.Receive().(type) {
		case redis.Message:
			c <- v.Data
		}
	}
}
