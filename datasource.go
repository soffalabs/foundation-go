package sf

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"strings"
)

type DbLinkCallback = func(ds DbLink) error

type TenantsLoaderFn = func() ([]string, error)
type DbLink struct {
	ServiceName   string
	Name          string
	Url           string
	TenantsLoader TenantsLoaderFn
	connection    *gorm.DB
	Migrations    []*gormigrate.Migration
}

type DatasourceLoader interface {
	LoadDatasources() ([]DbLink, error)
}

type FixedDatasourceLoader struct {
	DatasourceLoader
	Items []DbLink
}

type TenantsLoader struct {
	DatasourceLoader
	Db    DbLink
	Query string
}

// *********************************************************************************************************************
// Operations
// *********************************************************************************************************************

func (ds *DbLink) Migrate(schema *string) error {

	if ds.Migrations == nil {
		log.Warn("[%s] no migrations found to apply.", ds.Name)
		return nil
	}

	AssertNotEmpty(ds.ServiceName, "Datasource serviceName is required")
	AssertNotEmpty(ds.Name, "Datasource name is required")

	if schema != nil {
		return ds.internalMigrations(ds.ServiceName, ds.Migrations, schema)
	} else if ds.TenantsLoader != nil {
		log.Info("Factory datasource found, scanning all schemas")
		items, err := ds.TenantsLoader()
		if err != nil {
			return err
		}
		for _, sc := range items {
			if err = ds.internalMigrations(ds.ServiceName, ds.Migrations, &sc); err != nil {
				return err
			}
		}
		return nil
	} else {
		return ds.internalMigrations(ds.ServiceName, ds.Migrations, nil)
	}
}

func (tl FixedDatasourceLoader) LoadDatasources() ([]DbLink, error) {
	return tl.Items, nil
}

func (ds *DbLink) bootstrap() error {
	if ds.connection != nil {
		return nil
	}
	if h.IsStrEmpty(ds.Url) {
		return errors.Errorf("invalid databaseUrl provided (empty)")
	}
	cnx, err := dburl.Parse(ds.Url)
	if err != nil {
		return errors.Errorf("error parsing databaseUrl: %v", err)
	}

	var dialect gorm.Dialector
	if cnx.Driver == "sqlite3" {
		dialect = sqlite.Open(cnx.DSN)
	} else if cnx.Driver == "postgres" {
		dialect = postgres.Open(cnx.DSN)
	} else {
		return errors.Errorf("Unsupported database dialect: %s", cnx.Driver)
	}
	link, err := gorm.Open(dialect, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: ds.ServiceName + "_",
		},
	})
	if err != nil {
		return err
	}
	ds.connection = link
	return nil
}

func (ds *DbLink) Create(model interface{}) error {
	return ds.connection.Create(model).Error
}

func (ds *DbLink) Ping() error {
	return ds.connection.Exec("SELECT 1").Error
}

func (ds *DbLink) Save(model interface{}) error {
	return ds.connection.Save(model).Error
}

func (ds *DbLink) Exec(command string) error {
	return ds.connection.Exec(command).Error
}

func (ds *DbLink) QueryFirst(dest interface{}, query string, args ...interface{}) (bool, error) {
	result := ds.connection.Limit(1).Where(query, args...).Find(dest)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (ds *DbLink) First(model interface{}) error {
	return ds.connection.First(model).Error
}

func (ds *DbLink) Query(dest interface{}, opts *QueryOpts, where string, values ...interface{}) error {
	q := ds.connection.Model(dest).Where(where, values)
	if opts != nil && opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	return q.Find(dest).Error
}

func (ds *DbLink) Pluck(table string, column string, dest interface{}) error {
	return ds.connection.Table(ds.TableName(table)).Pluck(column, dest).Error
}

func (ds *DbLink) TableName(table string) string {
	prefix := ds.ServiceName + "_"
	if strings.HasPrefix(table, prefix) {
		return table
	}
	return prefix + table
}

func (ds *DbLink) Count(model interface{}) (int64, error) {
	var count int64
	if h := ds.connection.Model(model).Count(&count); h.Error != nil {
		return 0, h.Error
	}
	return count, nil
}

func (ds *DbLink) CreateSchema(name string) error {
	dialect := ds.connection.Dialector.Name()
	if "postgres" == dialect {
		if result := ds.connection.Exec(fmt.Sprintf("CREATE SCHEMA %s", name)); result.Error != nil {
			return result.Error
		}
	} else {
		log.Warnf("Schema creation not supported by: %s", dialect)
	}
	return nil
}

func (ds *DbLink) Transactional(callback func(link DbLink) error) error {
	return ds.connection.Transaction(func(tx *gorm.DB) error {
		txDS := ds
		txDS.connection = tx
		return callback(*txDS)
	})
}

func (ds *DbLink) FindAll(dest interface{}, limit int) error {
	tx := ds.connection.Limit(limit).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (ds *DbLink) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if h := ds.connection.Model(model).Where(where, args...).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}

func (ds *DbLink) UseSchema(name string) error {
	if ds.connection.Dialector.Name() == "sqlite" {
		return nil
	}
	if res := ds.connection.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (ds DbLink) WithTenant(name string, cb func(ds DbLink) error) error {
	return ds.Transactional(func(link DbLink) error {
		if err := link.UseSchema(name); err != nil {
			return err
		}
		return cb(link)
	})
}

func (ds *DbLink) internalMigrations(prefix string, migrations []*gormigrate.Migration, schema *string) error {

	if migrations == nil {
		log.Infof("[%s] no migrationss found to apply", ds.Name)
		return nil
	}

	if err := ds.bootstrap(); err != nil {
		return err
	}

	return ds.connection.Transaction(func(tx *gorm.DB) error {
		if schema != nil && tx.Dialector.Name() != "sqlite" {
			if res := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", *schema)); res.Error != nil {
				return res.Error
			}
			if res := tx.Exec(fmt.Sprintf("SET search_path to %s", *schema)); res.Error != nil {
				return res.Error
			}
		}
		tableName := fmt.Sprintf("%s_%s", strings.ReplaceAll(prefix, "-", "_"), gormigrate.DefaultOptions.TableName)
		m := gormigrate.New(tx, &gormigrate.Options{
			TableName:                 tableName,
			IDColumnName:              gormigrate.DefaultOptions.IDColumnName,
			IDColumnSize:              gormigrate.DefaultOptions.IDColumnSize,
			UseTransaction:            gormigrate.DefaultOptions.UseTransaction,
			ValidateUnknownMigrations: gormigrate.DefaultOptions.ValidateUnknownMigrations,
		}, migrations)
		if err := m.Migrate(); err != nil {
			return errors.Errorf("[%s] could not be migrated -- %v", ds.Name, err)
		} else {
			log.Infof("[%s] migrations applied successfully", ds.Name)
			return nil
		}
	})

}
