package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm/clause"
)

type TenantSecretStore struct {
	*Store
}

// NewTenantSecretStore executes the store.NewTenantSecretStore operation.
func NewTenantSecretStore(s *Store) *TenantSecretStore {
	return &TenantSecretStore{Store: s}
}

// List executes the store.TenantSecretStore.List operation.
func (s *TenantSecretStore) List(ctx context.Context, companyID uuid.UUID) ([]domain.TenantSecret, error) {
	applog.TraceCall(ctx, "store.TenantSecretStore.List")
	var secrets []domain.TenantSecret
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).Order("kind ASC").Find(&secrets).Error; err != nil {
		return nil, err
	}
	return secrets, nil
}

// Get executes the store.TenantSecretStore.Get operation.
func (s *TenantSecretStore) Get(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (*domain.TenantSecret, error) {
	applog.TraceCall(ctx, "store.TenantSecretStore.Get")
	var secret domain.TenantSecret
	if err := s.db.WithContext(ctx).Where("company_id = ? AND kind = ?", companyID, kind).First(&secret).Error; err != nil {
		return nil, err
	}
	return &secret, nil
}

// Upsert executes the store.TenantSecretStore.Upsert operation.
func (s *TenantSecretStore) Upsert(ctx context.Context, companyID uuid.UUID, secret *domain.TenantSecret) error {
	applog.TraceCall(ctx, "store.TenantSecretStore.Upsert")
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

// Delete executes the store.TenantSecretStore.Delete operation.
func (s *TenantSecretStore) Delete(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) error {
	applog.TraceCall(ctx, "store.TenantSecretStore.Delete")
	return s.db.WithContext(ctx).Where("company_id = ? AND kind = ?", companyID, kind).Delete(&domain.TenantSecret{}).Error
}
