package soffa_core

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbLinkCallback = func(ds DataSource) error

type DataSource struct {
	Name          string
	Url           string
	TenantsLoader func() ([]string, error)
	connection    *gorm.DB
	Migrations    []*gormigrate.Migration
}

type DatasourceLoader interface {
	LoadDatasources() ([]DataSource, error)
}

type FixedDatasourceLoader struct {
	DatasourceLoader
	Items []DataSource
}

type TenantsLoader struct {
	DatasourceLoader
	Db    DataSource
	Query string
}


// *********************************************************************************************************************
// Operations
// *********************************************************************************************************************

func (ds DataSource) Migrate(schema *string) error {
	if schema != nil {
		return ds.internalMigrations(ds.Migrations, schema)
	} else if ds.TenantsLoader != nil {
		log.Info("Factory datasource found, scanning all schemas")
		items, err := ds.TenantsLoader()
		if err != nil {
			return err
		}
		for _, sc := range items {
			if err = ds.internalMigrations(ds.Migrations, &sc); err != nil {
				return err
			}
		}
		return nil
	} else {
		return ds.internalMigrations(ds.Migrations, nil)
	}
}

func (tl FixedDatasourceLoader) LoadDatasources() ([]DataSource, error) {
	return tl.Items, nil
}

func (ds *DataSource) bootstrap() error {
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
	link, err := gorm.Open(dialect, &gorm.Config{})
	if err != nil {
		return err
	}
	ds.connection = link
	return nil
}

func (ds DataSource) Create(model interface{}) error {
	return ds.connection.Create(model).Error
}

func (ds DataSource) Ping() error {
	return ds.connection.Exec("SELECT 1").Error
}

func (ds DataSource) Save(model interface{}) error {
	return ds.connection.Save(model).Error
}

func (ds DataSource) Exec(command string) error {
	return ds.connection.Exec(command).Error
}

func (ds DataSource) QueryFirst(dest interface{}, query string, args ...interface{}) (bool, error) {
	result := ds.connection.Limit(1).Where(query, args...).Find(dest)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (ds DataSource) First(model interface{}) error {
	return ds.connection.First(model).Error
}

func (ds DataSource) Query(dest interface{}, opts *QueryOpts, where string, values ...interface{}) error {
	q := ds.connection.Model(dest).Where(where, values)
	if opts != nil && opts.Limit>0 {
		q = q.Limit(opts.Limit)
	}
	return q.Find(dest).Error
}

func (ds DataSource) Pluck(table string, column string, dest interface{}) error {
	return ds.connection.Table(table).Pluck(column, dest).Error
}

func (ds DataSource) Count(model interface{}) (int64, error) {
	var count int64
	if h := ds.connection.Model(model).Count(&count); h.Error != nil {
		return 0, h.Error
	}
	return count, nil
}

func (ds DataSource) CreateSchema(name string) error {
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

func (ds DataSource) Transactional(callback func(link DataSource) error) error {
	return ds.connection.Transaction(func(tx *gorm.DB) error {
		txDS := ds
		txDS.connection = tx
		return callback(txDS)
	})
}

func (ds DataSource) FindAll(dest interface{}, limit int) error {
	tx := ds.connection.Limit(limit).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (ds DataSource) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if h := ds.connection.Model(model).Where(where, args...).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}

func (ds DataSource) UseSchema(name string) error {
	if res := ds.connection.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (ds DataSource) internalMigrations(migrations []*gormigrate.Migration, schema *string) error {

	if migrations == nil {
		log.Info("[%s] no migrationss found to apply", ds.Name)
		return nil
	}

	return ds.connection.Transaction(func(tx *gorm.DB) error {
		if schema != nil  && tx.Dialector.Name() != "sqlite" {
			if res := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", *schema)); res.Error != nil {
				return res.Error
			}
			if res := tx.Exec(fmt.Sprintf("SET search_path to %s", *schema)); res.Error != nil {
				return res.Error
			}
		}
		m := gormigrate.New(tx, gormigrate.DefaultOptions, migrations)
		if err := m.Migrate(); err != nil {
			return fmt.Errorf("[%s] could not be migrated -- %v", ds.Name, err)
		} else {
			log.Infof("[%s] migrations applied successfully", ds.Name)
			return nil
		}
	})

}
