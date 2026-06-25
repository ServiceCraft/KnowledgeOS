package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type PricingStore struct {
	*Store
}

// NewPricingStore executes the store.NewPricingStore operation.
func NewPricingStore(s *Store) *PricingStore {
	return &PricingStore{Store: s}
}

// List executes the store.PricingStore.List operation.
func (s *PricingStore) List(ctx context.Context, companyID uuid.UUID, filter domain.PricingNodeFilter) ([]domain.PricingNode, int64, error) {
	applog.TraceCall(ctx, "store.PricingStore.List")
	var items []domain.PricingNode
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.PricingNode{}).Scopes(tenantScope(companyID))

	if filter.ParentID != nil {
		q = q.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.NodeType != "" {
		q = q.Where("node_type = ?", filter.NodeType)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("name ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID executes the store.PricingStore.GetByID operation.
func (s *PricingStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.PricingNode, error) {
	applog.TraceCall(ctx, "store.PricingStore.GetByID")
	var item domain.PricingNode
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create executes the store.PricingStore.Create operation.
func (s *PricingStore) Create(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	applog.TraceCall(ctx, "store.PricingStore.Create")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		node.SyncVersion = seq
		node.SyncOrigin = s.origin
		node.CompanyID = companyID
		return tx.Create(node).Error
	})
}

// Update executes the store.PricingStore.Update operation.
func (s *PricingStore) Update(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	applog.TraceCall(ctx, "store.PricingStore.Update")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		node.SyncVersion = seq
		node.SyncOrigin = s.origin
		return tx.Model(node).Scopes(tenantScope(companyID)).Where("id = ?", node.ID).Updates(node).Error
	})
}

// Delete executes the store.PricingStore.Delete operation.
func (s *PricingStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.PricingStore.Delete")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.PricingNode{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

// ListSince executes the store.PricingStore.ListSince operation.
func (s *PricingStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.PricingNode, error) {
	applog.TraceCall(ctx, "store.PricingStore.ListSince")
	var items []domain.PricingNode
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ApplyRemote executes the store.PricingStore.ApplyRemote operation.
func (s *PricingStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	applog.TraceCall(ctx, "store.PricingStore.ApplyRemote")
	node.SyncOrigin = "cloud"
	node.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", node.ID, companyID).
		Assign(node).FirstOrCreate(node).Error
}
