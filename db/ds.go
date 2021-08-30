package db

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DS struct {
	order         int
	Id            string
	Url           string
	TablePrefix   string
	Migrations    []*gormigrate.Migration
	TenantsLoader TenantsLoader
	link          *Link
}

func (ds *DS) ping() error {
	return ds.link.Ping()
}

func (ds *DS) migrate() {
	ds.migrateSchema("")
}

func (ds *DS) migrateSchema(schema string) {
	if ds.Migrations == nil {
		log.Default.Warn("[%s] no migrations found to apply.", ds.Id)
		return
	}
	if !h.IsEmpty(schema) {
		log.Default.Infof("migrating schema %s", schema)
		ds.internalMigrations(ds.Migrations, schema)
	} else if ds.TenantsLoader != nil {
		log.Default.Info("multitenant datasource found, scanning all schemas")
		items := ds.TenantsLoader()
		if items == nil || len(items) ==0 {
			log.Default.Warn("empty tenants list received, skipping migrations")
		} else {
			for _, sc := range items {
				log.Default.Infof("applying migrations on schema %s", sc)
				ds.internalMigrations(ds.Migrations, sc)
			}
		}
	} else {
		ds.internalMigrations(ds.Migrations, "")
	}
}

func (ds *DS) internalMigrations(migrations []*gormigrate.Migration, schema string) {

	if migrations == nil {
		errors.RaiseNew("[%s] no migrationss found to apply", ds.Id)
	}

	ds.link.Transactional(func(tx *Link) {
		if !h.IsEmpty(schema) && tx.supportsSchemas() {
			tx.createSchemas(schema)
			tx.UseSchema(schema)
		}
		gormLink := tx.base.(*GormLink)
		m := gormigrate.New(gormLink.conn, &gormigrate.Options{
			TableName:                 ds.TablePrefix + gormigrate.DefaultOptions.TableName,
			IDColumnName:              gormigrate.DefaultOptions.IDColumnName,
			IDColumnSize:              gormigrate.DefaultOptions.IDColumnSize,
			UseTransaction:            gormigrate.DefaultOptions.UseTransaction,
			ValidateUnknownMigrations: gormigrate.DefaultOptions.ValidateUnknownMigrations,
		}, migrations)

		errors.Raise(m.Migrate())
		log.Default.Infof("[%s] migrations applied successfully", ds.Id)
	})

}

func (ds *DS) bootstrap() {
	if ds.link != nil {
		return
	}
	if h.IsStrEmpty(ds.Url) {
		errors.RaiseNew("invalid databaseUrl provided (empty)")
	}
	cnx, err := dburl.Parse(ds.Url)
	if err != nil {
		errors.Raisef(err, "error parsing databaseUrl: %s", ds.Url)
	}

	var dialect gorm.Dialector
	if cnx.Driver == "sqlite3" {
		dialect = sqlite.Open(cnx.DSN)
	} else if cnx.Driver == "postgres" {
		dialect = postgres.Open(cnx.DSN)
	} else {
		errors.RaiseNew("Unsupported database dialect: %s", cnx.Driver)
	}
	link, err := gorm.Open(dialect, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: ds.TablePrefix,
		},
	})
	if err != nil {
		errors.Raisef(err, "conection to datasource %s failed", ds.Url)
	}
	ds.link = &Link{&GormLink{conn: link, ds: ds}}
}
