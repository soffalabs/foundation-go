package broker

import (
	"github.com/soffa-io/soffa-core-go/counters"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"strings"
)

var (
	SendMessageCounter   = counters.NewCounter("x_sys_broker_send_message", "Will track messages sent", true)
	MessageHandleCounter = counters.NewCounter("x_sys_broker_handle_message", "Will track messages received", true)
)

type Message struct {
	Event string
	Data  []byte
}

type Event struct {
	Event string
	Data  interface{}
}

type Handler = func(msg Message) interface{}

type Client interface {
	Start()
	Ping() error
	Publish(subject string, data interface{}) error
	Request(subject string, data interface{}, dest interface{}) error
	Subscribe(subject string, handler Handler)
}

func NewClient(url string, name string) Client {
	if strings.HasPrefix(url, "nats://") {
		return newNatsMessageClient(url, name)
	} else if url == "mock" {
		return NewMockClient(name)
	}
	log.Default.Fatal(errors.Errorf("unsupported broker url: %s (try nats:// or mock for tests)", url))
	return nil
}

func (m Message) Decode(dest interface{}) error {
	return h.DecodeBytes(m.Data, dest)
}

type Manager struct {
	client Client
}

func (c Manager) Subscribe(event string, handler Handler) Manager {
	c.client.Subscribe(event, handler)
	return c
}
