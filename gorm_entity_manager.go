package soffa

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GormEntityManager struct {
	EntityManager
	Name       string
	Link       *gorm.DB
	migrations []*gormigrate.Migration
}

func (em GormEntityManager) Create(model interface{}) error {
	if result := em.Link.Create(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (em GormEntityManager) GetBy(dest interface{}, query string, args ...interface{}) error {
	if result := em.Link.Where(query, args...).First(dest); result.Error != nil {
		return result.Error
	}
	return nil
}

func (em GormEntityManager) First(model interface{}) error {
	if result := em.Link.First(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (em GormEntityManager) Count(model interface{}) (int64, error) {
	var count int64
	if h := em.Link.Model(model).Count(&count); h.Error != nil {
		return 0, h.Error
	}
	return count, nil
}

func (em GormEntityManager) CreateSchema(name string) error {
	dialect := em.Link.Dialector.Name()
	if "postgres" == dialect {
		if result := em.Link.Exec(fmt.Sprintf("CREATE SCHEMA %s", name)); result.Error != nil {
			return result.Error
		}
	} else {
		log.Warnf("Schema creation not supported by: %s", dialect)
	}
	return nil
}

func (em GormEntityManager) Transactional(callback func(em EntityManager) error) error {
	return em.Link.Transaction(func(tx *gorm.DB) error {
		return callback(GormEntityManager{Link: tx})
	})
}

func (em GormEntityManager) FindAll(dest interface{}, limit int) error {
	tx := em.Link.Limit(limit).Find(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (em GormEntityManager) ExistsBy(model interface{}, where string, args ...interface{}) (bool, error) {
	var count int64
	if h := em.Link.Model(model).Where(where, args...).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}

func (em GormEntityManager) ApplyMigrations() error {
	if em.migrations != nil {
		m := gormigrate.New(em.Link, gormigrate.DefaultOptions, em.migrations)
		if err := m.Migrate(); err != nil {
			return fmt.Errorf("[%s] could not be migrated -- %v", em.Name, err)
		} else {
			log.Printf("[%s] migrations applied successfully", em.Name)
		}
	} else {
		log.Infof("[%s] no migrationss found to apply", em.Name)
	}
	return nil
}
