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

func (link *GormLink) MigrateTenant(schema string) {
	link.ds.migrateSchema(schema)
}

func (link *GormLink) Migrate() {
	link.ds.migrate()
}

func (link *GormLink) WithTenant(tenant string) BaseLink {
	return &GormLink{
		conn:   link.conn,
		ds:     link.ds,
		tenant: tenant,
	}
}

func (link *GormLink) Ping() error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Exec("SELECT 1").Error
	})

}

func (link *GormLink) Create(model interface{}) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Create(model).Error
	})
}

func (link *GormLink) Save(model interface{}) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Save(model).Error
	})
}

func (link *GormLink) Exec(command string) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Exec(command).Error
	})
}

func (link *GormLink) First(model interface{}) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.First(model).Error
	})
}

func (link *GormLink) Pluck(table interface{}, column string, dest interface{}) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Model(table).Pluck(column, dest).Error
	})
}

func (link *GormLink) Count(model interface{}) (int64, error) {
	var count int64 = 0
	err := link.withConn(func(conn *gorm.DB) error {
		return conn.Model(model).Count(&count).Error
	})
	return count, err
}

func (link *GormLink) CreateSchema(name string) error {
	return link.withConn(func(conn *gorm.DB) error {
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

func (link *GormLink) Transactional(callback func(link BaseLink) error) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Transaction(func(tx *gorm.DB) error {
			link := &GormLink{conn: tx, ds: link.ds}
			return callback(link)
		})
	})
}

func (link *GormLink) Find(dest interface{}, query Query) Result {

	res := &Result{}

	res.Error = link.withConn(func(conn *gorm.DB) error {
		exec := conn.Offset(query.offset).Limit(query.limit).Order(query.sort)
		if query.whereMap != nil {
			exec = exec.Where(h.UnwrapMap(*query.whereMap))
		} else if !h.IsEmpty(query.where) {
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

func (link *GormLink) Truncate(model interface{}) error {
	return link.withConn(func(conn *gorm.DB) error {
		return conn.Delete(model, "1=1").Error
	})
}

func (link *GormLink) ExistsById(model interface{}, id string) (bool, error) {
	return link.ExistsBy(model, "id = ?", id)
}

func (link *GormLink) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var out = false
	err := link.withConn(func(conn *gorm.DB) error {
		var count int64
		if res := conn.Model(model).Where(where, args...).Count(&count); res.Error != nil {
			return res.Error
		}
		out = count > 0
		return nil
	})
	return out, err
}

func (link *GormLink) UseSchema(name string) error {
	if !link.supportsSchemas() {
		return nil
	}
	if res := link.conn.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (link *GormLink) supportsSchemas() bool {
	return link.conn.Dialector.Name() != "sqlite"
}

func (link *GormLink) createSchemas(names ...string) error {
	for _, name := range names {
		if res := link.conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", name)); res.Error != nil {
			return res.Error
		}
	}
	return nil

}

// ------------------------------------------------------------------------------------------------

func (link *GormLink) withConn(cb func(tx *gorm.DB) error) error {
	var err error
	if h.IsEmpty(link.tenant) {
		err = cb(link.conn)
	} else {
		_, exists := link.conn.Get("tenant_active")
		if exists {
			err = cb(link.conn)
		} else {
			err = link.conn.Transaction(func(tx *gorm.DB) error {
				tx.Set("tenant_active", true)
				if link.supportsSchemas() {
					if res := tx.Exec(fmt.Sprintf("SET search_path to %s", link.tenant)); res.Error != nil {
						return res.Error
					}
				}
				return cb(tx)
			})
		}
	}
	link.ds.counterOperations.Record(err)
	if err != nil {
		log.Default.With("tenant", link.tenant).Error(err)
	}
	return err

}
