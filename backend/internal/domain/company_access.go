package domain

import (
	"context"

	"github.com/google/uuid"
)

// CompanyExistenceChecker reports whether a company record exists.
type CompanyExistenceChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
