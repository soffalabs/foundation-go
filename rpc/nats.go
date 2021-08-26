package rpc

import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/commons"
	"github.com/soffa-io/soffa-core-go/log"
	"time"
)


type BinaryMessage struct {
	Data  []byte
	Reply func([]byte)
}

type BinaryMessageHandler = func(event BinaryMessage) error

type Client struct {
	nc *nats.Conn
}

func (n *Client) Publish(subj string, data []byte) error {
	return n.nc.Publish(subj, data)
}

func (n *Client) Request(subj string, payload interface{}) ([]byte, error) {
	// bytes, err := prepareMessage(event, payload)
	bytes, err := commons.GetBytes(payload)
	if err != nil {
		log.Errorf("[nats] bytes encoding failed -- %v", subj, err)
		return nil, err
	}
	msg, err := n.nc.Request(subj, bytes, 10*time.Second)
	if err != nil {
		log.Errorf("[nats] error sending message to %s -- %v", subj, err)
		return nil, err
	}

	log.Infof("[nats] message sent to to %s", subj)
	return msg.Data, nil
}

func (n *Client) Subscribe(subject string, handler BinaryMessageHandler) {
	_, err := n.nc.Subscribe(subject, func(m *nats.Msg) {
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
		log.Error(handler(bmsg))
	})
	log.FatalErr(err)
	log.Infof("[nats] subscribred to %s", subject)
}

func ConnectToNats(url string, name string) *Client {
	nc, err := nats.Connect(url)
	log.FatalErr(err)
	log.Infof("[nats] %s is now connected to %s", name, url)
	return &Client{
		nc: nc,
	}
}
