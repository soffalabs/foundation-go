package broker

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"strings"
)

type Message struct {
	Event string
	Data []byte
}


type Event struct {
	Event string
	Data interface{}
}

type Handler  = func(msg Message) interface{}

type Client interface {
	Ping() error
	Publish(subject string, data interface{}) error
	Request(subject string, data interface{}, dest interface{}) error
	Subscribe(subject string, handler Handler)
}


func NewClient(url string, name string) Client {
	if strings.HasPrefix(url, "nats://") {
		return newNatsMessageClient(url, name)
	}else if url == "mock" {
		return NewMockClient(name)
	}
	log.Fatal(errors.Errorf("unsupported broker url: %s (try nats:// or mock for tests)", url))
	return nil
}


func (m Message) Decode(dest interface{}) error {
	return h.DecodeBytes(m.Data, dest)
}

type Manager struct {
	client Client
}

func NewClientWrapper(client Client) Manager {
	return Manager{client: client}
}

func (c Manager) Subscribe(event string, handler Handler) Manager {
	c.client.Subscribe(event, handler)
	return c
}