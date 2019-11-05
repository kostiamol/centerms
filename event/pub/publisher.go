package pub

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kostiamol/centerms/cfg"
	"github.com/satori/go.uuid"

	"github.com/kostiamol/centerms/log"

	gproto "github.com/golang/protobuf/proto"
	"github.com/kostiamol/centerms/proto"
	"github.com/nats-io/go-nats"
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

func (p *publisher) Publish(mac, data string) error {
	var (
		err          error
		conn         *nats.Conn
		retryAttempt uint32
	)

	for {
		conn, err = nats.Connect(fmt.Sprintf("%s:%d", p.addr.Host, p.addr.Port))
		if err != nil && retryAttempt < p.retryAttempts {
			p.log.Error("pubNewCfgPatchEvent(): nats connectivity status is DISCONNECTED")
			retryAttempt++
			duration := time.Duration(rand.Intn(int(p.retryTimeout.Seconds())))
			time.Sleep(time.Second*duration + 1)
			continue
		}
		break
	}
	defer conn.Close()

	e := &proto.Event{
		AggregateId:   mac,
		AggregateType: "cfg_svc",
		EventId:       uuid.NewV4().String(),
		EventType:     "cfg_patched",
		EventData:     data,
	}

	b, err := gproto.Marshal(e)
	if err != nil {
		return fmt.Errorf("Marshall(): %s", err)
	}

	topic := fmt.Sprintf("%s.%s", p.cfgPatchTopic, mac)
	if err := conn.Publish(topic, b); err != nil {
		return err
	}

	p.log.With("func", "pubNewCfgPatchEvent", "event", log.EventCfgPatchCreated).
		Infof("cfg patch [%s] for device with ID [%s]", data, mac)

	return nil
}
