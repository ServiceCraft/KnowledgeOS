package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserMembershipReader resolves company assignments for a user.
type UserMembershipReader interface {
	GetCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// GetMembershipVersion returns the user's current membership version, which is
	// bumped whenever company assignments change. It is used to invalidate access
	// tokens issued before an access change.
	GetMembershipVersion(ctx context.Context, userID uuid.UUID) (int, error)
}
