package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type UserStore struct {
	*Store
}

func NewUserStore(s *Store) *UserStore {
	return &UserStore{Store: s}
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) Create(ctx context.Context, user *domain.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *UserStore) Update(ctx context.Context, user *domain.User) error {
	return s.db.WithContext(ctx).Save(user).Error
}
