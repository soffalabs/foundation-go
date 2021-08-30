package db

import (
	"fmt"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"gorm.io/gorm"
)


type GormLink struct {
	BaseLink
	conn   *gorm.DB
	ds     *DS
	tenant string
}

func (ds *GormLink) MigrateTenant(schema string)  {
	 ds.ds.migrateSchema(schema)
}

func (ds *GormLink) Migrate()  {
	 ds.ds.migrate()
}

func (ds *GormLink) WithTenant(tenant string) BaseLink {
	return &GormLink{
		conn:   ds.conn,
		ds:     ds.ds,
		tenant: tenant,
	}
}

func (ds *GormLink) Ping() error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Exec("SELECT 1").Error
	})

}

func (ds *GormLink) Create(model interface{}) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Create(model).Error
	})
}

func (ds *GormLink) Save(model interface{}) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Save(model).Error
	})
}

func (ds *GormLink) Exec(command string) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Exec(command).Error
	})
}

func (ds *GormLink) First(model interface{}) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.First(model).Error
	})
}

func (ds *GormLink) Pluck(table interface{}, column string, dest interface{}) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Model(table).Pluck(column, dest).Error
	})
}

func (ds *GormLink) Count(model interface{}) (int64, error) {
	var count int64 = 0
	err := ds.withConn(func(conn *gorm.DB) error {
		return conn.Model(model).Count(&count).Error
	})
	return count, err
}

func (ds *GormLink) CreateSchema(name string) error {
	return ds.withConn(func(conn *gorm.DB) error {
		dialect := conn.Dialector.Name()
		if "postgres" == dialect {
			if result := conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", name)); result.Error != nil {
				return result.Error
			}
		} else {
			log.Default.Warnf("Schema creation not supported by: %s", dialect)
		}
		return nil
	})
}

func (ds *GormLink) Transactional(callback func(link BaseLink) error) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Transaction(func(tx *gorm.DB) error {
			link := &GormLink{conn: tx, ds: ds.ds}
			return callback(link)
		})
	})
}

func (ds *GormLink) Find(dest interface{}, query Query) Result {

	res := &Result{}

	res.Error = ds.withConn(func(conn *gorm.DB) error {
		exec := conn.Offset(query.offset).Limit(query.limit).Order(query.sort)
		if query.whereMap != nil {
			exec = exec.Where(h.UnwrapMap(*query.whereMap))
		}else if !h.IsEmpty(query.where) {
			exec = exec.Where(query.where, query.args...)
		}
		out := exec.Find(dest)
		if out.Error != nil {
			dest = nil
			return out.Error
		}

		res.RowsAffected = out.RowsAffected
		res.Empty = out.RowsAffected == 0

		if res.Empty {
			dest = nil
		}
		return nil
	})

	return *res
}

func (ds *GormLink) Truncate(model interface{}) error {
	return ds.withConn(func(conn *gorm.DB) error {
		return conn.Delete(model, "1=1").Error
	})
}

func (ds *GormLink) ExistsById(model interface{}, id string) (bool, error) {
	return ds.ExistsBy(model, "id = ?", id)
}

func (ds *GormLink) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var out = false
	err := ds.withConn(func(conn *gorm.DB) error {
		var count int64
		if res := conn.Model(model).Where(where, args...).Count(&count); res.Error != nil {
			return res.Error
		}
		out = count > 0
		return nil
	})
	return out, err
}

func (ds *GormLink) UseSchema(name string) error {
	if !ds.supportsSchemas() {
		return nil
	}
	if res := ds.conn.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (ds *GormLink) supportsSchemas() bool {
	return ds.conn.Dialector.Name() != "sqlite"
}

func (ds *GormLink) createSchemas(names ...string) error {
	for _, name := range names {
		if res := ds.conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", name)); res.Error != nil {
			return res.Error
		}
	}
	return nil

}


// ------------------------------------------------------------------------------------------------

func (ds *GormLink) withConn(cb func(tx *gorm.DB) error) error {
	if h.IsEmpty(ds.tenant) {
		return cb(ds.conn)
	}
	_, exists := ds.conn.Get("tenant_active")
	if exists {
		return cb(ds.conn)
	}
	return ds.conn.Transaction(func(tx *gorm.DB) error {
		tx.Set("tenant_active", true)
		if ds.supportsSchemas() {
			if res := tx.Exec(fmt.Sprintf("SET search_path to %s", ds.tenant)); res.Error != nil {
				return res.Error
			}
		}
		return cb(tx)
	})
}

