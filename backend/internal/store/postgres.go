package store

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store struct {
	db     *gorm.DB
	origin string
}

func New(db *gorm.DB, origin string) *Store {
	return &Store{db: db, origin: origin}
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func tenantScope(companyID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("company_id = ?", companyID)
	}
}

func paginate(page, limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit
		return db.Offset(offset).Limit(limit)
	}
}
