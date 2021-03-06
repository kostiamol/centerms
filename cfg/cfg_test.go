package cfg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	_, err := New()
	assert.NotNil(t, err)
}

func TestConfig(t *testing.T) {
	c := &Config{
		Service: Service{
			AppID:              "centerms",
			LogLevel:           "debug",
			RetryAttempts:      5,
			RetryTimeout:       time.Duration(100),
			PortRPC:            1111,
			PortREST:           2222,
			PortWebSocket:      3333,
			TerminationTimeout: 100,
		},
		TraceAgent: TraceAgent{
			Addr: Addr{
				Host: "localhost",
				Port: 4444},
		},
		Publisher: Publisher{
			Addr: Addr{
				Host: "localhost",
				Port: 4222},
			CfgPatchTopic: "topic"},
		Store: Store{
			Addr: Addr{
				Host: "localhost",
				Port: 6379},
			Password:         "password",
			IdleTimeout:      240,
			MaxIdlePoolConns: 5,
		},
		Token: Token{
			PublicKey:  "public_key",
			PrivateKey: "private_key",
		},
	}
	err := c.validate()
	assert.Nil(t, err)

	c = &Config{}
	err = c.validate()
	assert.NotNil(t, err)
}

func TestServiceConfig(t *testing.T) {
	svc := Service{
		AppID:              "centerms",
		LogLevel:           "debug",
		RetryAttempts:      5,
		RetryTimeout:       time.Duration(100),
		PortRPC:            1111,
		PortREST:           2222,
		PortWebSocket:      3333,
		TerminationTimeout: 3,
	}
	err := svc.validate()
	assert.Nil(t, err)

	svc = Service{}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms"}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug"}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryAttempts: 5}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryAttempts: 5,
		RetryTimeout: time.Duration(100)}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryAttempts: 5,
		RetryTimeout: time.Duration(100), PortRPC: 1111}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryAttempts: 5,
		RetryTimeout: time.Duration(100), PortRPC: 1111, PortREST: 2222,
		PortWebSocket: 3333}
	err = svc.validate()
	assert.NotNil(t, err)
}

func TestStoreCfg(t *testing.T) {
	s := Store{
		Addr:             Addr{Host: "localhost", Port: 1111},
		Password:         "password",
		IdleTimeout:      240,
		MaxIdlePoolConns: 5,
	}
	err := s.validate()
	assert.Nil(t, err)

	s = Store{}
	err = s.validate()
	assert.NotNil(t, err)

	s = Store{Addr: Addr{Host: "localhost"}}
	err = s.validate()
	assert.NotNil(t, err)

	s = Store{Addr: Addr{Host: "localhost", Port: 1111}}
	err = s.validate()
	assert.NotNil(t, err)
}

func TestTokenCfg(t *testing.T) {
	tkn := Token{PublicKey: "public_key", PrivateKey: "private_key"}
	err := tkn.Validate()
	assert.Nil(t, err)

	tkn = Token{}
	err = tkn.Validate()
	assert.NotNil(t, err)

	tkn = Token{PublicKey: "public_key"}
	err = tkn.Validate()
	assert.NotNil(t, err)
}

func TestPublisherCfg(t *testing.T) {
	n := Publisher{Addr: Addr{Host: "localhost", Port: 1111}, CfgPatchTopic: "topic"}
	err := n.validate()
	assert.Nil(t, err)

	n = Publisher{}
	err = n.validate()
	assert.NotNil(t, err)

	n = Publisher{Addr: Addr{Host: "localhost"}}
	err = n.validate()
	assert.NotNil(t, err)
}
