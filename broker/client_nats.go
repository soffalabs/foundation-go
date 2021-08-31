package broker

import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/soffa-io/soffa-core-go/sentry"
	"time"
)

type NatsMessageClient struct {
	Client
	id   string
	conn *nats.Conn
	open bool
	log  *log.Logger
}

func (n *NatsMessageClient) Ping() error {
	return nil
}

func (n *NatsMessageClient) Start() {
	n.open = true
}

func (n *NatsMessageClient) Publish(subj string, data interface{}) error {
	err := SendMessageCounter.Watch(func() error {
		if bytes, err := h.GetBytes(data); err != nil {
			return err
		}else {
			return n.conn.Publish(subj, bytes)
		}
	})
	sentry.CaptureException(err)
	return err
}

func (n *NatsMessageClient) Request(subj string, data interface{}, dest interface{}) error {
	err := SendMessageCounter.Watch(func() error {
		// bytes, err := prepareMessage(event, payload)
		n.log.Infof("requesting data from channel :%s", subj)
		bytes, err := h.GetBytes(data)
		if err != nil {
			n.log.Error(err)
			return errors.Wrapf(err, "[nats] bytes encoding failed -- %v", subj, err)
		}
		msg, err := n.conn.Request(subj, bytes, 10*time.Second)
		if err != nil {
			n.log.Error(err)
			return errors.Wrapf(err, "[nats] error sending message to %s -- %v", subj, err)
		}
		n.log.Infof("response received from channel %s", subj)
		return h.DecodeBytes(msg.Data, dest)
	})
	sentry.CaptureException(err)
	return err
}

func (n *NatsMessageClient) Subscribe(subj string, handler Handler) {

	loggger := n.log.With("broker.subject", subj)

	_, err := n.conn.Subscribe(subj, func(m *nats.Msg) {
		defer func() {
			re := recover()

			MessageHandleCounter.Recover(re, false)

			if re != nil {
				sentry.CaptureException(re.(error))
				loggger.Wrapf(re.(error), "[nats.%s] panic error received", subj)
				if m.Reply == "" {
					_ = m.Respond(nil)
				}
				_ = m.Nak()
			}
		}()

		loggger.Info("new message received")

		if !n.open {
			loggger.Warn("sending NACK before application is not yet ready to receive messages")
			_ = m.Nak()
			if m.Reply == "" {
				_ = m.Respond(nil)
			}
			return
		}

		if loggger.IsDebugEnabled() {
			loggger.Debugf("%s", m.Data)
		}

		bmsg := Message{Data: m.Data}
		response := handler(bmsg)

		if m.Reply == "" {
			_ = m.Ack()
		} else {
			bytes, err := h.GetBytes(response)
			if err != nil {
				loggger.Wrap(err, "error encoding data to send back")
				SendMessageCounter.Inc()
			} else {
				if err = m.Respond(bytes); err != nil {
					loggger.Wrapf(err, "error sending response to %s", m.Reply)
					SendMessageCounter.Inc()
				} else {
					loggger.Infof("data successfully sent back to %s", m.Reply)
				}
			}
		}
	})
	if err != nil {
		loggger.Fatal("unable to subscribe to subject")
	}
	loggger.Info("subscription is active")
}

func newNatsMessageClient(url string, name string) Client {
	log.Default.Infof("connecting to nats instance %s", url)
	nc, err := nats.Connect(url)
	if err != nil {
		log.Default.Fatal(errors.Wrapf(err, "error connecting to nats server: %s", url))
	}
	log.Default.Infof("application is now connected to nats server %s", url)
	return &NatsMessageClient{conn: nc, id: name, log: log.Default.With("broker", "nats", "broker.name", name)}
}
