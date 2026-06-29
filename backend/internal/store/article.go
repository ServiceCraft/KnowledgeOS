package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type ArticleStore struct {
	*Store
}

// NewArticleStore executes the store.NewArticleStore operation.
func NewArticleStore(s *Store) *ArticleStore {
	return &ArticleStore{Store: s}
}

// List executes the store.ArticleStore.List operation.
func (s *ArticleStore) List(ctx context.Context, companyID uuid.UUID, filter domain.ArticleFilter) ([]domain.Article, int64, error) {
	applog.TraceCall(ctx, "store.ArticleStore.List")
	var items []domain.Article
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.Article{}).Scopes(tenantScope(companyID))

	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		q = q.Where("title ILIKE ? OR body ILIKE ?", like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID executes the store.ArticleStore.GetByID operation.
func (s *ArticleStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Article, error) {
	applog.TraceCall(ctx, "store.ArticleStore.GetByID")
	var item domain.Article
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create executes the store.ArticleStore.Create operation.
func (s *ArticleStore) Create(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	applog.TraceCall(ctx, "store.ArticleStore.Create")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return applog.TraceErr(ctx, "article: bump sync sequence", err)
		}
		article.SyncVersion = seq
		article.SyncOrigin = s.origin
		article.CompanyID = companyID
		return tx.Create(article).Error
	})
}

// Update executes the store.ArticleStore.Update operation.
func (s *ArticleStore) Update(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	applog.TraceCall(ctx, "store.ArticleStore.Update")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return applog.TraceErr(ctx, "article: bump sync sequence", err)
		}
		article.SyncVersion = seq
		article.SyncOrigin = s.origin
		return tx.Model(article).Scopes(tenantScope(companyID)).Where("id = ?", article.ID).Updates(article).Error
	})
}

// Delete executes the store.ArticleStore.Delete operation.
func (s *ArticleStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.ArticleStore.Delete")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return applog.TraceErr(ctx, "article: bump sync sequence", err)
		}
		return tx.Model(&domain.Article{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

// ListSince executes the store.ArticleStore.ListSince operation.
func (s *ArticleStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.Article, error) {
	applog.TraceCall(ctx, "store.ArticleStore.ListSince")
	var items []domain.Article
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ApplyRemote executes the store.ArticleStore.ApplyRemote operation.
func (s *ArticleStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	applog.TraceCall(ctx, "store.ArticleStore.ApplyRemote")
	article.SyncOrigin = "cloud"
	article.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", article.ID, companyID).
		Assign(article).FirstOrCreate(article).Error
}
