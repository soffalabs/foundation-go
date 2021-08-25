package sf

import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/log"
)

type NatsClient struct {
	nc *nats.Conn
}

func (n *NatsClient) Publish(subj string,  data []byte) error {
	return n.nc.Publish(subj, data)
}

func (n *NatsClient) Send(subj string, event string, payload interface{}) error {
	bytes, err := prepareMessage(event, payload)
	if err != nil {
		log.Errorf("[nats] error sending message to %s -- %v", subj, err)
		return err
	}
	if log.IsDebugEnabled() {
		log.Debugf("[nats] publishing data %s", bytes)
		log.Debugf("[nats] publishing data %s", event)
	}
	err = n.nc.Publish(subj, bytes)
	if err != nil {
		log.Errorf("[nats] error sending message to %s -- %v", subj, err)
		return err
	}

 	log.Infof("[nats] message sent to to %s", subj)
	return nil
}

func (n *NatsClient) Subscribe(subject string, handler MessageHandler) error {
	_, err := n.nc.Subscribe(subject, func(m *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		}()
		if log.IsDebugEnabled() {
			log.Debugf("[nats] message received %s", subject)
		}
		msg, err := DecodeMessage(m.Data)
		_ = Capture("nats.received.error.decode", err)
		if err == nil {
			msg.Reply = func(data interface{}) error {
				bytes, err := ToJson(data)
				if err != nil {
					return err
				}
				return m.Respond(bytes)
			}
			err = handler(*msg)
			_ = Capture("data.encoding.bytes", err)
		}
	})
	log.Infof("[nats] subscribred to %s", subject)
	return err
}

func ConnectToNats(url string, name string) *NatsClient {
	nc, err := nats.Connect(url)
	Fatal(err)
	log.Infof("[nats] %s is now connected to %s", name, url)
	return &NatsClient{
		nc: nc,
	}
}
