package broker

import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"time"
)

type NatsMessageClient struct {
	Client
	id   string
	conn *nats.Conn
}

func (n *NatsMessageClient) Ping() error {
	return nil
}

func (n *NatsMessageClient) Publish(subj string, data interface{}) error {
	bytes, err := h.GetBytes(data)
	if err != nil {
		return err
	}
	err = n.conn.Publish(subj, bytes)
	return err
}

func (n *NatsMessageClient) Request(subj string, data interface{}, dest interface{}) error {
	// bytes, err := prepareMessage(event, payload)
	bytes, err := h.GetBytes(data)
	if err != nil {
		return errors.Wrapf(err, "[nats] bytes encoding failed -- %v", subj, err)
	}
	msg, err := n.conn.Request(subj, bytes, 10*time.Second)
	if err != nil {
		return errors.Wrapf(err, "[nats] error sending message to %s -- %v", subj, err)
	}
	log.Infof("[nats] message sent to to %s", subj)
	return h.DecodeBytes(msg.Data, dest)
}

func (n *NatsMessageClient) Subscribe(subj string, handler Handler) {
	_, err := n.conn.Subscribe(subj, func(m *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
				_ = m.Respond(nil)
				_ = m.Nak()
			}
		}()
		if log.IsDebugEnabled() {
			log.Debugf("[nats] message received %s", subj)
		}
		bmsg := Message{Data: m.Data}
		response := handler(bmsg)

		if m.Reply == "" {
			_ = m.Ack()
		} else {
			bytes, err := h.GetBytes(response)
			errors.Raise(err)
			errors.Raise(m.Respond(bytes))
		}
	})
	log.FatalIf(err)
	log.Infof("[nats] subscribred to %s", subj)
}

func newNatsMessageClient(url string, name string) Client {
	nc, err := nats.Connect(url)
	log.FatalIf(err)
	log.Infof("[nats] %s is now connected to %s", name, url)
	return &NatsMessageClient{conn: nc, id: name}
}
