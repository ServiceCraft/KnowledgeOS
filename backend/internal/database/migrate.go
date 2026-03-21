package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB, migrationsDir string) error {
	db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		var count int64
		db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", f).Scan(&count)
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return fmt.Errorf("execute migration %s: %w", f, err)
			}
			return tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", f).Error
		})
		if err != nil {
			return err
		}
		log.Printf("Applied migration: %s", f)
	}

	return nil
}
