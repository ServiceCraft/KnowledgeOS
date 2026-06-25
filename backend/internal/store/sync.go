package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SyncStore struct {
	*Store
}

// NewSyncStore executes the store.NewSyncStore operation.
func NewSyncStore(s *Store) *SyncStore {
	return &SyncStore{Store: s}
}

// GetSequence executes the store.SyncStore.GetSequence operation.
func (s *SyncStore) GetSequence(ctx context.Context, companyID uuid.UUID) (*domain.SyncSequence, error) {
	applog.TraceCall(ctx, "store.SyncStore.GetSequence")
	var seq domain.SyncSequence
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&seq).Error; err != nil {
		return nil, err
	}
	return &seq, nil
}

// InitSequence executes the store.SyncStore.InitSequence operation.
func (s *SyncStore) InitSequence(ctx context.Context, companyID uuid.UUID) error {
	applog.TraceCall(ctx, "store.SyncStore.InitSequence")
	return s.db.WithContext(ctx).Exec(
		"INSERT INTO sync_sequence (company_id, current_seq) VALUES (?, 0) ON CONFLICT DO NOTHING", companyID,
	).Error
}

// GetWatermark executes the store.SyncStore.GetWatermark operation.
func (s *SyncStore) GetWatermark(ctx context.Context, companyID uuid.UUID) (*domain.SyncWatermark, error) {
	applog.TraceCall(ctx, "store.SyncStore.GetWatermark")
	var wm domain.SyncWatermark
	err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&wm).Error
	if err != nil {
		wm = domain.SyncWatermark{CompanyID: companyID}
		return &wm, nil
	}
	return &wm, nil
}

// UpdateWatermark executes the store.SyncStore.UpdateWatermark operation.
func (s *SyncStore) UpdateWatermark(ctx context.Context, wm *domain.SyncWatermark) error {
	applog.TraceCall(ctx, "store.SyncStore.UpdateWatermark")
	return s.db.WithContext(ctx).Exec(
		`INSERT INTO sync_watermarks (company_id, last_local_seq, last_cloud_seq, last_sync_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (company_id) DO UPDATE SET
		   last_local_seq = EXCLUDED.last_local_seq,
		   last_cloud_seq = EXCLUDED.last_cloud_seq,
		   last_sync_at = EXCLUDED.last_sync_at`,
		wm.CompanyID, wm.LastLocalSeq, wm.LastCloudSeq, wm.LastSyncAt,
	).Error
}

// GetStatus executes the store.SyncStore.GetStatus operation.
func (s *SyncStore) GetStatus(ctx context.Context, companyID uuid.UUID) (*domain.SyncStatus, error) {
	applog.TraceCall(ctx, "store.SyncStore.GetStatus")
	var st domain.SyncStatus
	err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&st).Error
	if err != nil {
		st = domain.SyncStatus{CompanyID: companyID}
		return &st, nil
	}
	return &st, nil
}

// UpdateStatus executes the store.SyncStore.UpdateStatus operation.
func (s *SyncStore) UpdateStatus(ctx context.Context, status *domain.SyncStatus) error {
	applog.TraceCall(ctx, "store.SyncStore.UpdateStatus")
	return s.db.WithContext(ctx).Exec(
		`INSERT INTO sync_status (company_id, last_sync_at, last_sync_result, last_error, subscription_active)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (company_id) DO UPDATE SET
		   last_sync_at = EXCLUDED.last_sync_at,
		   last_sync_result = EXCLUDED.last_sync_result,
		   last_error = EXCLUDED.last_error,
		   subscription_active = EXCLUDED.subscription_active`,
		status.CompanyID, status.LastSyncAt, status.LastSyncResult, status.LastError, status.SubscriptionActive,
	).Error
}

// CreateLog executes the store.SyncStore.CreateLog operation.
func (s *SyncStore) CreateLog(ctx context.Context, entry *domain.SyncLog) error {
	applog.TraceCall(ctx, "store.SyncStore.CreateLog")
	return s.db.WithContext(ctx).Create(entry).Error
}

// Auth token methods

func (s *SyncStore) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	applog.TraceCall(ctx, "store.SyncStore.SaveRefreshToken")
	return s.db.WithContext(ctx).Create(token).Error
}

// GetRefreshToken executes the store.SyncStore.GetRefreshToken operation.
func (s *SyncStore) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	applog.TraceCall(ctx, "store.SyncStore.GetRefreshToken")
	var token domain.RefreshToken
	if err := s.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken executes the store.SyncStore.RevokeRefreshToken operation.
func (s *SyncStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	applog.TraceCall(ctx, "store.SyncStore.RevokeRefreshToken")
	return s.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllUserTokens executes the store.SyncStore.RevokeAllUserTokens operation.
func (s *SyncStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	applog.TraceCall(ctx, "store.SyncStore.RevokeAllUserTokens")
	return s.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// GetAPIKeyByHash executes the store.SyncStore.GetAPIKeyByHash operation.
func (s *SyncStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	applog.TraceCall(ctx, "store.SyncStore.GetAPIKeyByHash")
	var key domain.APIKey
	if err := s.db.WithContext(ctx).Where("key_hash = ? AND revoked_at IS NULL", keyHash).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// CreateAPIKey executes the store.SyncStore.CreateAPIKey operation.
func (s *SyncStore) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {
	applog.TraceCall(ctx, "store.SyncStore.CreateAPIKey")
	return s.db.WithContext(ctx).Create(key).Error
}

// TouchAPIKey executes the store.SyncStore.TouchAPIKey operation.
func (s *SyncStore) TouchAPIKey(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.SyncStore.TouchAPIKey")
	return s.db.WithContext(ctx).Model(&domain.APIKey{}).Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}
