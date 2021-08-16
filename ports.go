package soffa

import (
	log "github.com/sirupsen/logrus"
)

const FakeAmqpurl = "@faker"

type MessagePublisher interface {
	Send(channel string, message Message) error
}

type MessageHandler interface {
	HandleMessage(event Message) error
}

type EntityManager interface {
	Create(model interface{}) error
	Transactional(callback func(em EntityManager) error) error
	FindAll(dest interface{}, limit int) error
	FindBy(dest interface{}, where string, args ...interface{}) error
	ExistsBy(model interface{}, where string, args ...interface{}) (bool, error)
	First(model interface{}) error
	CreateSchema(name string) error
	Count(model interface{}) (int64, error)
	GetBy(dest interface{}, query string, args ...interface{}) error
}

type FakeMessagePublisherImpl struct {
	MessagePublisher
}

func (p FakeMessagePublisherImpl) Send(channel string, message Message) error {
	log.Info("[FakerPublisher] Message sent to channel: %s", channel)
	return nil
}
