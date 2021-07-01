package pub

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kostiamol/centerms/cfg"
	"github.com/satori/go.uuid"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/proto"
	"github.com/nats-io/go-nats"
	gproto "google.golang.org/protobuf/proto"
)

type (
	Cfg struct {
		Addr          cfg.Addr
		CfgPatchTopic string
		Log           log.Logger
		RetryTimeout  time.Duration
		RetryAttempts uint32
	}

	publisher struct {
		addr          cfg.Addr
		cfgPatchTopic string
		log           log.Logger
		retryTimeout  time.Duration
		retryAttempts uint32
	}
)

func New(c *Cfg) *publisher { // nolint
	return &publisher{
		addr:          c.Addr,
		cfgPatchTopic: c.CfgPatchTopic,
		log:           c.Log,
		retryTimeout:  c.RetryTimeout,
		retryAttempts: c.RetryAttempts,
	}
}

func (p *publisher) Publish(id, data string) error {
	var (
		err          error
		conn         *nats.Conn
		retryAttempt uint32
	)

	for {
		conn, err = nats.Connect(fmt.Sprintf("%s:%d", p.addr.Host, p.addr.Port))
		if err != nil && retryAttempt < p.retryAttempts {
			p.log.Error("func pubNewCfgPatchEvent: nats connectivity status is DISCONNECTED")
			retryAttempt++
			duration := time.Duration(rand.Intn(int(p.retryTimeout.Seconds())))
			time.Sleep(time.Second*duration + 1)
			continue
		}
		break
	}
	defer conn.Close()

	e := &proto.Event{
		AggregateId:   id,
		AggregateType: "cfg_svc",
		EventId:       uuid.NewV4().String(),
		EventType:     "cfg_patched",
		EventData:     data,
	}

	b, err := gproto.Marshal(e)
	if err != nil {
		return fmt.Errorf("func Marshall: %s", err)
	}

	topic := fmt.Sprintf("%s.%s", p.cfgPatchTopic, id)
	if err := conn.Publish(topic, b); err != nil {
		return err
	}

	p.log.With("event", log.EventCfgPatchCreated, "patch", data, "devID", id)

	return nil
}
