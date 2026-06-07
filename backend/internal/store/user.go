package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type UserStore struct {
	*Store
}

func NewUserStore(s *Store) *UserStore {
	return &UserStore{Store: s}
}

func (s *UserStore) List(ctx context.Context, companyID uuid.UUID, filter domain.UserFilter) ([]domain.User, int64, error) {
	var items []domain.User
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.User{}).Where("company_id = ?", companyID)

	if filter.Query != "" {
		q = q.Where("email ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Role != nil {
		q = q.Where("role = ?", *filter.Role)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "created_at DESC"
	switch filter.Sort {
	case "created_at":
		order = "created_at ASC"
	case "-created_at", "":
		order = "created_at DESC"
	}

	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order(order).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
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

// Update persists the mutable fields of a user. Save would also overwrite the
// password hash with the value present on the struct, so callers are expected to
// pass a fully-loaded user record.
func (s *UserStore) Update(ctx context.Context, user *domain.User) error {
	return s.db.WithContext(ctx).Save(user).Error
}

func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.User{}).Error
}

// ErrNotFound mirrors gorm's record-not-found for callers that want to detect it
// without importing gorm directly.
var ErrNotFound = gorm.ErrRecordNotFound
