package db

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
)

type TenantsLoader = func() []string

type BaseLink interface {
	MigrateTenant(schema string)
	Migrate()
	WithTenant(tenant string) BaseLink
	Ping() error
	Create(model interface{}) error
	Save(model interface{}) error
	Exec(command string) error
	Pluck(table interface{}, column string, dest interface{}) error
	Count(model interface{}) (int64, error)
	CreateSchema(name string) error
	Find(dest interface{}, query Query) Result
	Transactional(callback func(link BaseLink) error) error
	Truncate(model interface{}) error
	ExistsById(model interface{}, id string) (bool, error)
	ExistsBy(model interface{}, where string, args ...interface{}) (bool, error)
	UseSchema(name string) error
	supportsSchemas() bool
	createSchemas(schemas ...string) error
}

type Link struct {
	ds   *DS
	base BaseLink
}

func (l *Link) MigrateTenant(schema string) {
	l.base.MigrateTenant(schema)
}

func (l *Link) Migrate() {
	l.base.Migrate()
}

func (l *Link) Ping() error {
	return l.base.Ping()
}

func (l *Link) Create(model interface{}) {
	errors.Raise(l.base.Create(model))
}

func (l *Link) Save(model interface{}) {
	errors.Raise(l.base.Save(model))
}

func (l *Link) Exec(command string) {
	errors.Raise(l.base.Exec(command))
}

func (l *Link) Pluck(table interface{}, column string, dest interface{}) {
	errors.Raise(l.base.Pluck(table, column, dest))
}

func (l *Link) Count(model interface{}) int64 {
	res, err := l.base.Count(model)
	errors.Raise(err)
	return res
}

func (l *Link) CreateSchema(name string) {
	errors.Raise(l.base.CreateSchema(name))
}

func (l *Link) Find(dest interface{}, query *Query) Result {
	res := l.base.Find(dest, *query)
	errors.Raise(res.Error)
	return res
}

func (l *Link) First(dest interface{}, query *Query) bool {
	query.Limit(1)
	res := l.base.Find(dest, *query)
	errors.Raise(res.Error)
	return !res.Empty
}

func (l *Link) FindById(dest interface{}, id string) bool {
	return l.First(dest, Q().W(h.Map{"id": id}))
}

func (l *Link) Truncate(model interface{}) {
	errors.Raise(l.base.Truncate(model))
}

func (l *Link) ExistsById(model interface{}, id string) bool {
	res, err := l.base.ExistsById(model, id)
	errors.Raise(err)
	return res
}

func (l *Link) ExistsBy(model interface{}, where string, args ...interface{}) bool {
	res, err := l.base.ExistsBy(model, where, args...)
	errors.Raise(err)
	return res
}

func (l *Link) UseSchema(name string) {
	errors.Raise(l.base.UseSchema(name))
}
func (l *Link) supportsSchemas() bool {
	return l.base.supportsSchemas()
}

func (l *Link) createSchemas(schemas ...string) {
	errors.Raise(l.base.createSchemas(schemas...))
}

func (l *Link) Transactional(callback func(link *Link)) {
	errors.Raise(l.base.Transactional(func(link BaseLink) error {
		callback(&Link{ds: l.ds, base: link})
		return nil
	}))
}

func (l *Link) Tenant(tenant string) *Link {
	return &Link{ds: l.ds, base: l.base.WithTenant(tenant)}
}
