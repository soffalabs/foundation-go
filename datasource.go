package sf

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"strings"
)

type DbLinkCallback = func(ds EntityManager) error

type EntityManager struct {
	ServiceName   string
	Name          string
	Url           string
	TenantsLoader func() ([]string, error)
	connection    *gorm.DB
	Migrations    []*gormigrate.Migration
}

type DatasourceLoader interface {
	LoadDatasources() ([]EntityManager, error)
}

type FixedDatasourceLoader struct {
	DatasourceLoader
	Items []EntityManager
}

type TenantsLoader struct {
	DatasourceLoader
	Db    EntityManager
	Query string
}

// *********************************************************************************************************************
// Operations
// *********************************************************************************************************************

func (ds EntityManager) Migrate(schema *string) error {

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

func (tl FixedDatasourceLoader) LoadDatasources() ([]EntityManager, error) {
	return tl.Items, nil
}

func (ds *EntityManager) bootstrap() error {
	if ds.connection != nil {
		return nil
	}
	if IsStrEmpty(ds.Url) {
		return fmt.Errorf("invalid databaseUrl provided (empty)")
	}
	cnx, err := dburl.Parse(ds.Url)
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

func (ds EntityManager) Create(model interface{}) error {
	return ds.connection.Create(model).Error
}

func (ds EntityManager) Ping() error {
	return ds.connection.Exec("SELECT 1").Error
}

func (ds EntityManager) Save(model interface{}) error {
	return ds.connection.Save(model).Error
}

func (ds EntityManager) Exec(command string) error {
	return ds.connection.Exec(command).Error
}

func (ds EntityManager) QueryFirst(dest interface{}, query string, args ...interface{}) (bool, error) {
	result := ds.connection.Limit(1).Where(query, args...).Find(dest)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (ds EntityManager) First(model interface{}) error {
	return ds.connection.First(model).Error
}

func (ds EntityManager) Query(dest interface{}, opts *QueryOpts, where string, values ...interface{}) error {
	q := ds.connection.Model(dest).Where(where, values)
	if opts != nil && opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	return q.Find(dest).Error
}

func (ds EntityManager) Pluck(table string, column string, dest interface{}) error {
	return ds.connection.Table(ds.TableName(table)).Pluck(column, dest).Error
}

func (ds EntityManager) TableName(table string) string {
	prefix := ds.ServiceName + "_"
	if strings.HasPrefix(table, prefix) {
		return table
	}
	return prefix + table
}

func (ds EntityManager) Count(model interface{}) (int64, error) {
	var count int64
	if h := ds.connection.Model(model).Count(&count); h.Error != nil {
		return 0, h.Error
	}
	return count, nil
}

func (ds EntityManager) CreateSchema(name string) error {
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

func (ds EntityManager) Transactional(callback func(link EntityManager) error) error {
	return ds.connection.Transaction(func(tx *gorm.DB) error {
		txDS := ds
		txDS.connection = tx
		return callback(txDS)
	})
}

func (ds EntityManager) FindAll(dest interface{}, limit int) error {
	tx := ds.connection.Limit(limit).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (ds EntityManager) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if h := ds.connection.Model(model).Where(where, args...).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}

func (ds *EntityManager) UseSchema(name string) error {
	if ds.connection.Dialector.Name() == "sqlite" {
		return nil
	}
	if res := ds.connection.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (ds EntityManager) WithTenant(name string, cb func(ds EntityManager) error) error {
	return ds.Transactional(func(link EntityManager) error {
		if err := link.UseSchema(name); err != nil {
			return err
		}
		return cb(link)
	})
}

func (ds EntityManager) internalMigrations(prefix string, migrations []*gormigrate.Migration, schema *string) error {

	if migrations == nil {
		log.Infof("[%s] no migrationss found to apply", ds.Name)
		return nil
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
			return fmt.Errorf("[%s] could not be migrated -- %v", ds.Name, err)
		} else {
			log.Infof("[%s] migrations applied successfully", ds.Name)
			return nil
		}
	})

}
