package store

import (
	"errors"

	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}
