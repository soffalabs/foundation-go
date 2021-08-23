package soffa_core

import (
	"github.com/soffa-io/soffa-core-go/log"
)


type MessagePublisher interface {
	Send(channel string, event string, payload interface{}) error
	SendSelf(event string, payload interface{}) error
}

type MessageHandler = func(event Message) error

type QueryOpts struct {
	First int
	Limit int
}


type FakeMessagePublisherImpl struct {
	MessagePublisher
}

func (p FakeMessagePublisherImpl) Send(channel string, message Message) error {
	log.Infof("[FakerPublisher] Message sent to channel: %s", channel)
	return nil
}
