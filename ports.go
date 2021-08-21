package sf

import (
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
)

const FakeAmqpurl = "mocked"

type MessagePublisher interface {
	Send(channel string, event string, payload interface{}) error
	SendSelf(event string, payload interface{}) error
}

type MessageHandler = func(event Message) error


type DbLink interface {
	Create(model interface{}) error
	Save(model interface{}) error
	Exec(raw string) error
	Transactional(callback func(link DbLink) error) error
	FindAll(dest interface{}, limit int) error
	ExistsBy(model interface{}, where string, args ...interface{}) (bool, error)
	First(model interface{}) error
	Query(dest interface{}, query string, args ...interface{}) error
	Pluck(table string, column string, dest interface{}) error
	CreateSchema(name string) error
	Count(model interface{}) (int64, error)
	QueryFirst(dest interface{}, query string, args ...interface{}) (bool, error)
	ApplyMigrations(migrations []*gormigrate.Migration, schema *string) error
	UseSchema(name string) error
	Ping() error
}

type FakeMessagePublisherImpl struct {
	MessagePublisher
}

func (p FakeMessagePublisherImpl) Send(channel string, message Message) error {
	log.Info("[FakerPublisher] Message sent to channel: %s", channel)
	return nil
}
