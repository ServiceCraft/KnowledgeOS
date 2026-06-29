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

// GetCompanyIDs returns company IDs assigned to the user.
func (s *UserStore) GetCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	applog.TraceCall(ctx, "store.UserStore.GetCompanyIDs")
	var ids []uuid.UUID
	err := s.db.WithContext(ctx).Model(&domain.UserCompany{}).
		Where("user_id = ?", userID).
		Pluck("company_id", &ids).Error
	return ids, err
}

// SetCompanyIDs replaces all company assignments for the user.
func (s *UserStore) SetCompanyIDs(ctx context.Context, userID uuid.UUID, companyIDs []uuid.UUID) error {
	applog.TraceCall(ctx, "store.UserStore.SetCompanyIDs")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&domain.UserCompany{}).Error; err != nil {
			return err
		}
		for _, cid := range companyIDs {
			row := domain.UserCompany{UserID: userID, CompanyID: cid}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// HasCompany reports whether the user is assigned to the company.
func (s *UserStore) HasCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error) {
	applog.TraceCall(ctx, "store.UserStore.HasCompany")
	var count int64
	err := s.db.WithContext(ctx).Model(&domain.UserCompany{}).
		Where("user_id = ? AND company_id = ?", userID, companyID).
		Count(&count).Error
	return count > 0, err
}

// ListCompaniesForUser returns companies the user is assigned to.
func (s *UserStore) ListCompaniesForUser(ctx context.Context, userID uuid.UUID) ([]domain.Company, error) {
	applog.TraceCall(ctx, "store.UserStore.ListCompaniesForUser")
	var companies []domain.Company
	err := s.db.WithContext(ctx).
		Joins("JOIN user_companies uc ON uc.company_id = companies.id").
		Where("uc.user_id = ?", userID).
		Order("companies.name ASC").
		Find(&companies).Error
	return companies, err
}

func (s *UserStore) attachCompanyIDs(ctx context.Context, users []domain.User) error {
	ids := make([]uuid.UUID, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	var links []domain.UserCompany
	if err := s.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&links).Error; err != nil {
		return err
	}
	byUser := make(map[uuid.UUID][]uuid.UUID, len(users))
	for _, l := range links {
		byUser[l.UserID] = append(byUser[l.UserID], l.CompanyID)
	}
	for i := range users {
		if users[i].Role == domain.RoleSuperadmin {
			continue
		}
		users[i].CompanyIDs = byUser[users[i].ID]
	}
	return nil
}

// ErrNotFound mirrors gorm's record-not-found for callers that want to detect it
// without importing gorm directly.
var ErrNotFound = gorm.ErrRecordNotFound
