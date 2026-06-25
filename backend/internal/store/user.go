package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type UserStore struct {
	*Store
}

// NewUserStore executes the store.NewUserStore operation.
func NewUserStore(s *Store) *UserStore {
	return &UserStore{Store: s}
}

// List executes the store.UserStore.List operation.
func (s *UserStore) List(ctx context.Context, companyID uuid.UUID, filter domain.UserFilter) ([]domain.User, int64, error) {
	applog.TraceCall(ctx, "store.UserStore.List")
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

// GetByID executes the store.UserStore.GetByID operation.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	applog.TraceCall(ctx, "store.UserStore.GetByID")
	var user domain.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail executes the store.UserStore.GetByEmail operation.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	applog.TraceCall(ctx, "store.UserStore.GetByEmail")
	var user domain.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create executes the store.UserStore.Create operation.
func (s *UserStore) Create(ctx context.Context, user *domain.User) error {
	applog.TraceCall(ctx, "store.UserStore.Create")
	return s.db.WithContext(ctx).Create(user).Error
}

// Update persists the mutable fields of a user. Save would also overwrite the
// password hash with the value present on the struct, so callers are expected to
// pass a fully-loaded user record.
func (s *UserStore) Update(ctx context.Context, user *domain.User) error {
	applog.TraceCall(ctx, "store.UserStore.Update")
	return s.db.WithContext(ctx).Save(user).Error
}

// Delete executes the store.UserStore.Delete operation.
func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.UserStore.Delete")
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.User{}).Error
}

// ErrNotFound mirrors gorm's record-not-found for callers that want to detect it
// without importing gorm directly.
var ErrNotFound = gorm.ErrRecordNotFound
