package cfg

import (
	"os"
	"reflect"
	"strconv"
	"time"
)

type (
	// Config holds the app config.
	Config struct {
		Service    Service
		Publisher  Publisher
		Store      Store
		Token      Token
		TraceAgent TraceAgent
	}

	// Addr holds remote server's host and port.
	Addr struct {
		Host string
		Port uint32
	}

	сonfiger interface {
		validate() error
	}
)

// New initializes configuration structure with environment variables and returns it.
func New() (*Config, error) {
	c := &Config{
		Service: Service{
			AppID:                     os.Getenv("APP_ID"),
			LogLevel:                  os.Getenv("LOG_LEVEL"),
			RetryAttempts:             uintEnv("RETRY_ATTEMPTS"),
			RetryTimeout:              time.Millisecond * time.Duration(uintEnv("RETRY_TIMEOUT_MS")),
			PortRPC:                   uintEnv("PORT_RPC"),
			PortREST:                  uintEnv("PORT_REST"),
			PortWebSocket:             uintEnv("PORT_WEBSOCKET"),
			RoutineTerminationTimeout: time.Millisecond * time.Duration(uintEnv("ROUTINE_TERMINATION_TIMEOUT_MS"))},
		TraceAgent: TraceAgent{
			Addr: Addr{
				Host: os.Getenv("TRACE_HOST"),
				Port: uintEnv("TRACE_PORT")},
		},
		Publisher: Publisher{
			Addr: Addr{
				Host: os.Getenv("PUB_HOST"),
				Port: uintEnv("PUB_PORT")},
			CfgPatchTopic: os.Getenv("PUB_CFG_PATCH_TOPIC")},
		Store: Store{
			Addr: Addr{
				Host: os.Getenv("STORE_HOST"),
				Port: uintEnv("STORE_PORT")},
			Password:         os.Getenv("STORE_PASSWORD"),
			IdleTimeout:      time.Millisecond * time.Duration(uintEnv("STORE_IDLE_TIMEOUT_MS")),
			MaxIdlePoolConns: uintEnv("STORE_MAX_IDLE_POOL_CONNS")},
		Token: Token{
			PublicKey:  os.Getenv("PUBLIC_KEY"),
			PrivateKey: os.Getenv("PRIVATE_KEY")}}

	err := c.validate()

	return c, err
}

func (c *Config) validate() error {
	v := reflect.ValueOf(c).Elem()
	baseConfigType := reflect.TypeOf((*сonfiger)(nil)).Elem()

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Type.Implements(baseConfigType) {
			if err := v.Field(i).Interface().(сonfiger).validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func uintEnv(env string) uint32 {
	v := os.Getenv(env)
	if v == "" {
		return 0
	}

	u, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint32(u)
}
