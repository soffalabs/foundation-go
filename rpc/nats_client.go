package rpc

import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"time"
)

type NatsClient struct {
	Client
	conn *nats.Conn
}

func (n *NatsClient) Publish(subj string, data []byte) error {
	return n.conn.Publish(subj, data)
}

func (n *NatsClient) Serve(op string, cb func(string, []byte) (interface{}, error)) {
	n.Subscribe(op, func(msg BinaryMessage) error {
		res, err := cb(op, msg.Data)
		if err != nil {
			msg.Reply(nil)
			return err
		} else {
			bytes, err := h.GetBytes(res)
			if err != nil {
				msg.Reply(nil)
				return err
			}
			msg.Reply(bytes)
			return nil
		}
	})
}

func (n *NatsClient) ServeAll(subjs []string, cb func(string, []byte) (interface{}, error)) {
	for _, sub := range subjs {
		n.Serve(sub, cb)
	}
}

func (n *NatsClient) Request(subj string, payload interface{}, dest interface{}) error {
	// bytes, err := prepareMessage(event, payload)
	bytes, err := h.GetBytes(payload)
	if err != nil {
		return errors.Wrapf(err, "[nats] bytes encoding failed -- %v", subj, err)
	}
	msg, err := n.conn.Request(subj, bytes, 10*time.Second)
	if err != nil {
		return errors.Wrapf(err, "[nats] error sending message to %s -- %v", subj, err)
	}

	log.Infof("[nats] message sent to to %s", subj)

	if msg.Data == nil {
		return nil
	}
	return h.DecodeBytes(msg.Data, dest)
}

func (n *NatsClient) Subscribe(subject string, handler BinaryMessageHandler) {
	_, err := n.conn.Subscribe(subject, func(m *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		}()
		if log.IsDebugEnabled() {
			log.Debugf("[nats] message received %s", subject)
		}
		bmsg := BinaryMessage{
			Data: m.Data,
			Reply: func(data []byte) {
				if err := m.Respond(data); err != nil {
					log.Error(err)
				}
			},
		}
		err := handler(bmsg)
		log.ErrorIf(err, "message handling failed [%s] -- %v", subject, err)
	})
	log.FatalErr(err)
	log.Infof("[nats] subscribred to %s", subject)
}

func NewNatsClient(url string, name string) *NatsClient {
	nc, err := nats.Connect(url)
	log.FatalErr(err)
	log.Infof("[nats] %s is now connected to %s", name, url)
	return &NatsClient{ conn: nc,}
}
