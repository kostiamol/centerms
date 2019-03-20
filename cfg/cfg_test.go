package cfg

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	_, err := NewConfig()
	assert.NotNil(t, err)
}

func TestConfig(t *testing.T) {
	c := &Config{
		Service: Service{
			AppID:         "centerms",
			LogLevel:      "debug",
			RetryNumber:   5,
			RetryTimeout:  time.Duration(100),
			PortRPC:       1111,
			PortREST:      2222,
			PortWebSocket: 3333,
		},
		Store: Store{
			Host:     os.Getenv("STORE_HOST"),
			Port:     uintEnv("STORE_PORT"),
			Password: os.Getenv("STORE_PASSWORD"),
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
		AppID:         "centerms",
		LogLevel:      "debug",
		RetryNumber:   5,
		RetryTimeout:  time.Duration(100),
		PortRPC:       1111,
		PortREST:      2222,
		PortWebSocket: 3333,
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

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryNumber: 5}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryNumber: 5, RetryTimeout: time.Duration(100)}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryNumber: 5, RetryTimeout: time.Duration(100), PortRPC: 1111}
	err = svc.validate()
	assert.NotNil(t, err)

	svc = Service{AppID: "centerms", LogLevel: "debug", RetryNumber: 5, RetryTimeout: time.Duration(100), PortRPC: 1111, PortREST: 2222}
	err = svc.validate()
	assert.NotNil(t, err)
}

func TestStoreConfig(t *testing.T) {
	s := Store{
		Host:     "localhost",
		Port:     1111,
		Password: "password",
	}
	err := s.validate()
	assert.Nil(t, err)

	s = Store{}
	err = s.validate()
	assert.NotNil(t, err)

	s = Store{Host: "localhost"}
	err = s.validate()
	assert.NotNil(t, err)

	s = Store{Host: "localhost", Port: 1111}
	err = s.validate()
	assert.NotNil(t, err)
}
