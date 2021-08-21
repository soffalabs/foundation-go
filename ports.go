package sf

import (
	"github.com/go-gormigrate/gormigrate/v2"
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

type DbLink interface {
	Create(model interface{}) error
	Save(model interface{}) error
	Exec(raw string) error
	Transactional(callback func(link DbLink) error) error
	FindAll(dest interface{}, limit int) error
	Query(dest interface{}, opts *QueryOpts, where string, args ...interface{}) error
	ExistsBy(model interface{}, where string, args ...interface{}) (bool, error)
	First(model interface{}) error
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
