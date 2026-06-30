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

	q := s.db.WithContext(ctx).Model(&domain.User{}).
		Joins("JOIN user_companies uc ON uc.user_id = users.id").
		Where("uc.company_id = ?", companyID)

	if filter.Query != "" {
		q = q.Where("users.email ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Role != nil {
		q = q.Where("users.role = ?", *filter.Role)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "users.created_at DESC"
	switch filter.Sort {
	case "created_at":
		order = "users.created_at ASC"
	case "-created_at", "":
		order = "users.created_at DESC"
	}

	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order(order).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	if len(items) > 0 {
		if err := s.attachCompanyIDs(ctx, items); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

// GetByID executes the store.UserStore.GetByID operation.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	applog.TraceCall(ctx, "store.UserStore.GetByID")
	var user domain.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, mapErr(err)
	}
	return &user, nil
}

// GetByEmail executes the store.UserStore.GetByEmail operation.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	applog.TraceCall(ctx, "store.UserStore.GetByEmail")
	var user domain.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, mapErr(err)
	}
	return &user, nil
}

// Create executes the store.UserStore.Create operation.
func (s *UserStore) Create(ctx context.Context, user *domain.User) error {
	applog.TraceCall(ctx, "store.UserStore.Create")
	return s.db.WithContext(ctx).Create(user).Error
}

// CreateWithCompanies creates a user and company assignments atomically.
func (s *UserStore) CreateWithCompanies(ctx context.Context, user *domain.User, companyIDs []uuid.UUID) error {
	applog.TraceCall(ctx, "store.UserStore.CreateWithCompanies")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if len(companyIDs) == 0 {
			return nil
		}
		rows := make([]domain.UserCompany, len(companyIDs))
		for i, cid := range companyIDs {
			rows[i] = domain.UserCompany{UserID: user.ID, CompanyID: cid}
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

// Update persists the mutable fields of a user.
func (s *UserStore) Update(ctx context.Context, user *domain.User) error {
	applog.TraceCall(ctx, "store.UserStore.Update")
	return s.db.WithContext(ctx).Model(user).Select(
		"Email", "PasswordHash", "Role", "IsActive", "UpdatedAt",
	).Updates(user).Error
}

// Delete executes the store.UserStore.Delete operation.
func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.UserStore.Delete")
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.User{}).Error
}
