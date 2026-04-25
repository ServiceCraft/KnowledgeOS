package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type CallStore struct {
	*Store
}

func NewCallStore(s *Store) *CallStore {
	return &CallStore{Store: s}
}

func (s *CallStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Call, error) {
	var item domain.Call
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *CallStore) Create(ctx context.Context, companyID uuid.UUID, call *domain.Call) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		call.SyncVersion = seq
		call.SyncOrigin = s.origin
		call.CompanyID = companyID
		return tx.Create(call).Error
	})
}

func (s *CallStore) Update(ctx context.Context, companyID uuid.UUID, call *domain.Call) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		call.SyncVersion = seq
		call.SyncOrigin = s.origin
		return tx.Model(call).Scopes(tenantScope(companyID)).Where("id = ?", call.ID).
			Select("title", "transcript", "occurred_at", "duration_sec", "audio_url", "external_id", "sync_version", "sync_origin", "updated_by").
			Updates(call).Error
	})
}

func (s *CallStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Call{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *CallStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, call *domain.Call) error {
	call.CompanyID = companyID
	if call.SyncOrigin == "" {
		call.SyncOrigin = "cloud"
	}
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", call.ID, companyID).
		Assign(call).FirstOrCreate(call).Error
}

func (s *CallStore) ListAll(ctx context.Context, companyID uuid.UUID) ([]domain.Call, error) {
	var items []domain.Call
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Order("occurred_at DESC NULLS LAST, created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
