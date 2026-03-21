package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type CommentStore struct {
	*Store
}

func NewCommentStore(s *Store) *CommentStore {
	return &CommentStore{Store: s}
}

func (s *CommentStore) List(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID, filter domain.CommentFilter) ([]domain.Comment, int64, error) {
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

func (s *CommentStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Comment, error) {
	var item domain.Comment
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *CommentStore) Create(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
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

func (s *CommentStore) Update(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
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

func (s *CommentStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Comment{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *CommentStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.Comment, error) {
	var items []domain.Comment
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CommentStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	comment.SyncOrigin = "cloud"
	comment.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", comment.ID, companyID).
		Assign(comment).FirstOrCreate(comment).Error
}
