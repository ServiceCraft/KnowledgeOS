package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

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
		if len(companyIDs) == 0 {
			return tx.Model(&domain.User{}).Where("id = ?", userID).
				UpdateColumn("membership_version", gorm.Expr("membership_version + 1")).Error
		}
		rows := make([]domain.UserCompany, len(companyIDs))
		for i, cid := range companyIDs {
			rows[i] = domain.UserCompany{UserID: userID, CompanyID: cid}
		}
		if err := tx.CreateInBatches(rows, 100).Error; err != nil {
			return err
		}
		return tx.Model(&domain.User{}).Where("id = ?", userID).
			UpdateColumn("membership_version", gorm.Expr("membership_version + 1")).Error
	})
}

// HasCompany reports whether the user is assigned to the company.
func (s *UserStore) HasCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error) {
	applog.TraceCall(ctx, "store.UserStore.HasCompany")
	var exists bool
	err := s.db.WithContext(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM user_companies WHERE user_id = ? AND company_id = ?)`,
		userID, companyID,
	).Scan(&exists).Error
	return exists, err
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
