package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

func createCompanyAdminUser(
	ctx context.Context,
	users domain.UserRepository,
	email, passwordHash string,
	companyID uuid.UUID,
) (*domain.User, error) {
	if _, err := users.GetByEmail(ctx, email); err == nil {
		return nil, conflict("a user with this email already exists")
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}
	if err := users.CreateWithCompanies(ctx, user, []uuid.UUID{companyID}); err != nil {
		if isUniqueViolation(err) {
			return nil, conflict("a user with this email already exists")
		}
		return nil, err
	}
	user.CompanyIDs = []uuid.UUID{companyID}
	return user, nil
}

func containsCompanyID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
