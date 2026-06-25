package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm/clause"
)

type TenantSecretStore struct {
	*Store
}

func NewTenantSecretStore(s *Store) *TenantSecretStore {
	return &TenantSecretStore{Store: s}
}

func (s *TenantSecretStore) List(ctx context.Context, companyID uuid.UUID) ([]domain.TenantSecret, error) {
	var secrets []domain.TenantSecret
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).Order("kind ASC").Find(&secrets).Error; err != nil {
		return nil, err
	}
	return secrets, nil
}

func (s *TenantSecretStore) Get(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (*domain.TenantSecret, error) {
	var secret domain.TenantSecret
	if err := s.db.WithContext(ctx).Where("company_id = ? AND kind = ?", companyID, kind).First(&secret).Error; err != nil {
		return nil, err
	}
	return &secret, nil
}

func (s *TenantSecretStore) Upsert(ctx context.Context, companyID uuid.UUID, secret *domain.TenantSecret) error {
	secret.CompanyID = companyID
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "company_id"}, {Name: "kind"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ciphertext",
			"nonce",
			"metadata",
			"updated_at",
		}),
	}).Create(secret).Error
}

func (s *TenantSecretStore) Delete(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) error {
	return s.db.WithContext(ctx).Where("company_id = ? AND kind = ?", companyID, kind).Delete(&domain.TenantSecret{}).Error
}
