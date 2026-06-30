package database

import (
	"github.com/knowledgeos/backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection via GORM and tunes the pool from config.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.PGMaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.PGMaxOpenConns)
	}
	if cfg.PGMaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.PGMaxIdleConns)
	}
	sqlDB.SetConnMaxLifetime(cfg.PGConnMaxLifetime())

	return db, nil
}
