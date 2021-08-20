package sf

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GormDbLink struct {
	DbLink
	Name       string
	Connection *gorm.DB
}

func (em GormDbLink) Create(model interface{}) error {
	return em.Connection.Create(model).Error
}

func (em GormDbLink) Save(model interface{}) error {
	return em.Connection.Save(model).Error
}

func (em GormDbLink) Exec(command string) error {
	return em.Connection.Exec(command).Error
}

func (em GormDbLink) QueryFirst(dest interface{}, query string, args ...interface{}) (bool, error) {
	result := em.Connection.Limit(1).Where(query, args...).Find(dest)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (em GormDbLink) First(model interface{}) error {
	return em.Connection.First(model).Error
}

func (em GormDbLink) Query(dest interface{}, query string, values ...interface{}) error {
	return em.Connection.Raw(query, values).Scan(&dest).Error
}

func (em GormDbLink) Pluck(table string, column string, dest interface{}) error {
	return em.Connection.Table(table).Pluck(column, dest).Error
}

func (em GormDbLink) Count(model interface{}) (int64, error) {
	var count int64
	if h := em.Connection.Model(model).Count(&count); h.Error != nil {
		return 0, h.Error
	}
	return count, nil
}

func (em GormDbLink) CreateSchema(name string) error {
	dialect := em.Connection.Dialector.Name()
	if "postgres" == dialect {
		if result := em.Connection.Exec(fmt.Sprintf("CREATE SCHEMA %s", name)); result.Error != nil {
			return result.Error
		}
	} else {
		log.Warnf("Schema creation not supported by: %s", dialect)
	}
	return nil
}

func (em GormDbLink) Transactional(callback func(link DbLink) error) error {
	return em.Connection.Transaction(func(tx *gorm.DB) error {
		return callback(GormDbLink{Connection: tx})
	})
}

func (em GormDbLink) FindAll(dest interface{}, limit int) error {
	tx := em.Connection.Limit(limit).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (em GormDbLink) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if h := em.Connection.Model(model).Where(where, args...).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}

func (em GormDbLink) UseSchema(name string) error {
	if res := em.Connection.Exec(fmt.Sprintf("SET search_path to %s", name)); res.Error != nil {
		return res.Error
	}
	return nil
}

func (em GormDbLink) ApplyMigrations(migrations []*gormigrate.Migration, schema *string) error {

	if migrations == nil {
		log.Infof("[%s] no migrationss found to apply", em.Name)
		return nil
	}

	return em.Connection.Transaction(func(tx *gorm.DB) error {
		if schema != nil {
			if res := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", *schema)); res.Error != nil {
				return res.Error
			}
			if res := tx.Exec(fmt.Sprintf("SET search_path to %s", *schema)); res.Error != nil {
				return res.Error
			}
		}
		m := gormigrate.New(tx, gormigrate.DefaultOptions, migrations)
		if err := m.Migrate(); err != nil {
			return fmt.Errorf("[%s] could not be migrated -- %v", em.Name, err)
		} else {
			log.Printf("[%s] migrations applied successfully", em.Name)
			return nil
		}
	})

}
