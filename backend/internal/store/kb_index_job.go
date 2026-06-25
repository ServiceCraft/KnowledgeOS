package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type KBIndexJobStore struct {
	*Store
}

// NewKBIndexJobStore executes the store.NewKBIndexJobStore operation.
func NewKBIndexJobStore(s *Store) *KBIndexJobStore {
	return &KBIndexJobStore{Store: s}
}

// Enqueue executes the store.KBIndexJobStore.Enqueue operation.
func (s *KBIndexJobStore) Enqueue(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID, operation domain.KBIndexOperation) error {
	applog.TraceCall(ctx, "store.KBIndexJobStore.Enqueue")
	job := domain.KBIndexJob{
		CompanyID:   companyID,
		EntityType:  entityType,
		EntityID:    entityID,
		Operation:   operation,
		Status:      domain.KBIndexJobPending,
		AvailableAt: time.Now(),
	}
	return s.db.WithContext(ctx).Exec(`
		INSERT INTO kb_index_jobs (company_id, entity_type, entity_id, operation, status, available_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (company_id, entity_type, entity_id, operation)
		WHERE status IN ('pending', 'processing')
		DO UPDATE SET
			status = 'pending',
			available_at = now(),
			last_error = NULL,
			updated_at = now()
	`, job.CompanyID, job.EntityType, job.EntityID, job.Operation, job.Status, job.AvailableAt).Error
}

// ClaimPending executes the store.KBIndexJobStore.ClaimPending operation.
func (s *KBIndexJobStore) ClaimPending(ctx context.Context, limit int) ([]domain.KBIndexJob, error) {
	applog.TraceCall(ctx, "store.KBIndexJobStore.ClaimPending")
	if limit <= 0 {
		limit = 10
	}
	var jobs []domain.KBIndexJob
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ids []uuid.UUID
		if err := tx.Raw(`
			SELECT id
			FROM kb_index_jobs
			WHERE status = 'pending' AND available_at <= now()
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		`, limit).Scan(&ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := tx.Model(&domain.KBIndexJob{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     domain.KBIndexJobProcessing,
				"attempts":   gorm.Expr("attempts + 1"),
				"updated_at": gorm.Expr("now()"),
			}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Order("created_at ASC").Find(&jobs).Error
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// MarkDone executes the store.KBIndexJobStore.MarkDone operation.
func (s *KBIndexJobStore) MarkDone(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.KBIndexJobStore.MarkDone")
	return s.db.WithContext(ctx).Model(&domain.KBIndexJob{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     domain.KBIndexJobDone,
			"last_error": nil,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

// MarkFailed executes the store.KBIndexJobStore.MarkFailed operation.
func (s *KBIndexJobStore) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, nextAttempt time.Time, maxAttempts int) error {
	applog.TraceCall(ctx, "store.KBIndexJobStore.MarkFailed")
	statusExpr := "pending"
	if maxAttempts > 0 {
		statusExpr = "CASE WHEN attempts >= ? THEN 'failed' ELSE 'pending' END"
		return s.db.WithContext(ctx).Model(&domain.KBIndexJob{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":       gorm.Expr(statusExpr, maxAttempts),
				"last_error":   errMsg,
				"available_at": nextAttempt,
				"updated_at":   gorm.Expr("now()"),
			}).Error
	}
	return s.db.WithContext(ctx).Model(&domain.KBIndexJob{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       statusExpr,
			"last_error":   errMsg,
			"available_at": nextAttempt,
			"updated_at":   gorm.Expr("now()"),
		}).Error
}

// Stats executes the store.KBIndexJobStore.Stats operation.
func (s *KBIndexJobStore) Stats(ctx context.Context, companyID uuid.UUID) (map[domain.KBIndexJobStatus]int64, error) {
	applog.TraceCall(ctx, "store.KBIndexJobStore.Stats")
	type row struct {
		Status domain.KBIndexJobStatus
		Total  int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&domain.KBIndexJob{}).
		Select("status, COUNT(*) AS total").
		Where("company_id = ?", companyID).
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	stats := map[domain.KBIndexJobStatus]int64{
		domain.KBIndexJobPending:    0,
		domain.KBIndexJobProcessing: 0,
		domain.KBIndexJobDone:       0,
		domain.KBIndexJobFailed:     0,
	}
	for _, row := range rows {
		stats[row.Status] = row.Total
	}
	return stats, nil
}
