package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type CommentStore struct {
	*Store
}

// NewCommentStore executes the store.NewCommentStore operation.
func NewCommentStore(s *Store) *CommentStore {
	return &CommentStore{Store: s}
}

// List executes the store.CommentStore.List operation.
func (s *CommentStore) List(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID, filter domain.CommentFilter) ([]domain.Comment, int64, error) {
	applog.TraceCall(ctx, "store.CommentStore.List")
	var items []domain.Comment
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.Comment{}).Scopes(tenantScope(companyID)).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID executes the store.CommentStore.GetByID operation.
func (s *CommentStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Comment, error) {
	applog.TraceCall(ctx, "store.CommentStore.GetByID")
	var item domain.Comment
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create executes the store.CommentStore.Create operation.
func (s *CommentStore) Create(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	applog.TraceCall(ctx, "store.CommentStore.Create")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		comment.SyncVersion = seq
		comment.SyncOrigin = s.origin
		comment.CompanyID = companyID
		return tx.Create(comment).Error
	})
}

// Update executes the store.CommentStore.Update operation.
func (s *CommentStore) Update(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	applog.TraceCall(ctx, "store.CommentStore.Update")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		comment.SyncVersion = seq
		comment.SyncOrigin = s.origin
		return tx.Model(comment).Scopes(tenantScope(companyID)).Where("id = ?", comment.ID).Updates(comment).Error
	})
}

// Delete executes the store.CommentStore.Delete operation.
func (s *CommentStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.CommentStore.Delete")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Comment{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

// ListSince executes the store.CommentStore.ListSince operation.
func (s *CommentStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.Comment, error) {
	applog.TraceCall(ctx, "store.CommentStore.ListSince")
	var items []domain.Comment
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ApplyRemote executes the store.CommentStore.ApplyRemote operation.
func (s *CommentStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	applog.TraceCall(ctx, "store.CommentStore.ApplyRemote")
	comment.SyncOrigin = "cloud"
	comment.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", comment.ID, companyID).
		Assign(comment).FirstOrCreate(comment).Error
}
