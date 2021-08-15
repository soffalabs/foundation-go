package soffa

import (
	"gorm.io/gorm"
)

type GormEntityManager struct {
	EntityManager
	Link *gorm.DB
}

func (em GormEntityManager) Create(model interface{}) error {
	if result := em.Link.Create(model); result.Error != nil {
		return result.Error
	}
	return nil
}

func (em GormEntityManager) Transactional(callback func() error) error {
	return em.Link.Transaction(func(tx *gorm.DB) error {
		return callback()
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
	if h := em.Link.Model(model).Where(where, args).Count(&count); h.Error != nil {
		return false, h.Error
	}
	return count > 0, nil
}
