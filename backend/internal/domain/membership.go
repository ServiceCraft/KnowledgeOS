package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserMembershipReader resolves company assignments for a user.
type UserMembershipReader interface {
	GetCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
