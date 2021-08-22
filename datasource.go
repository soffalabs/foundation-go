package sf

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbLinkCallback = func(em DbLink) error

type DataSource struct {
	Name          string
	Url           string
	TenantsLoader func() ([]string, error)
	Link          DbLink
	Migrations    []*gormigrate.Migration
}

func (d DataSource) ApplyMigrations(schema *string) error {
	if schema != nil {
		return d.Link.ApplyMigrations(d.Migrations, schema)
	} else if d.TenantsLoader != nil {
		log.Info("Factory datasource found, scanning all schemas")
		items, err := d.TenantsLoader()
		if err != nil {
			return err
		}
		for _, sc := range items {
			if err = d.Link.ApplyMigrations(d.Migrations, &sc); err != nil {
				return err
			}
		}
		return nil
	} else {
		return d.Link.ApplyMigrations(d.Migrations, nil)
	}
}

func (d DataSource) Ping() error {
	return d.Link.Ping()
}

type DatasourceLoader interface {
	LoadDatasources() ([]DataSource, error)
}

type FixedDatasourceLoader struct {
	DatasourceLoader
	Items []DataSource
}

type DatabaseTenantsLoader struct {
	DatasourceLoader
	Db    DbLink
	Query string
}

func (tl FixedDatasourceLoader) LoadDatasources() ([]DataSource, error) {
	return tl.Items, nil
}

func (d *DataSource) Init() error {
	if d.Link != nil {
		return nil
	}
	if IsStrEmpty(d.Url) {
		return fmt.Errorf("invalid databaseUrl provided (empty)")
	}
	cnx, err := dburl.Parse(d.Url)
	if err != nil {
		return fmt.Errorf("error parsing databaseUrl: %v", err)
	}

	var dialect gorm.Dialector
	if cnx.Driver == "sqlite3" {
		dialect = sqlite.Open(cnx.DSN)
	} else if cnx.Driver == "postgres" {
		dialect = postgres.Open(cnx.DSN)
	} else {
		return fmt.Errorf("Unsupported database dialect: %s", cnx.Driver)
	}
	link, err := gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		return err
	}
	d.Link = GormDbLink{Name: d.Name, Connection: link}
	return nil
}
