package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SyncStore struct {
	*Store
}

func NewSyncStore(s *Store) *SyncStore {
	return &SyncStore{Store: s}
}

func (s *SyncStore) GetSequence(ctx context.Context, companyID uuid.UUID) (*domain.SyncSequence, error) {
	var seq domain.SyncSequence
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&seq).Error; err != nil {
		return nil, err
	}
	return &seq, nil
}

func (s *SyncStore) InitSequence(ctx context.Context, companyID uuid.UUID) error {
	return s.db.WithContext(ctx).Exec(
		"INSERT INTO sync_sequence (company_id, current_seq) VALUES (?, 0) ON CONFLICT DO NOTHING", companyID,
	).Error
}

func (s *SyncStore) GetWatermark(ctx context.Context, companyID uuid.UUID) (*domain.SyncWatermark, error) {
	var wm domain.SyncWatermark
	err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&wm).Error
	if err != nil {
		wm = domain.SyncWatermark{CompanyID: companyID}
		return &wm, nil
	}
	return &wm, nil
}

func (s *SyncStore) UpdateWatermark(ctx context.Context, wm *domain.SyncWatermark) error {
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

func (s *SyncStore) GetStatus(ctx context.Context, companyID uuid.UUID) (*domain.SyncStatus, error) {
	var st domain.SyncStatus
	err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&st).Error
	if err != nil {
		st = domain.SyncStatus{CompanyID: companyID}
		return &st, nil
	}
	return &st, nil
}

func (s *SyncStore) UpdateStatus(ctx context.Context, status *domain.SyncStatus) error {
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

func (s *SyncStore) CreateLog(ctx context.Context, entry *domain.SyncLog) error {
	return s.db.WithContext(ctx).Create(entry).Error
}

// Auth token methods

func (s *SyncStore) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	return s.db.WithContext(ctx).Create(token).Error
}

func (s *SyncStore) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	if err := s.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *SyncStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return s.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", time.Now()).Error
}

func (s *SyncStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func (s *SyncStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var key domain.APIKey
	if err := s.db.WithContext(ctx).Where("key_hash = ? AND revoked_at IS NULL", keyHash).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (s *SyncStore) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {
	return s.db.WithContext(ctx).Create(key).Error
}

func (s *SyncStore) TouchAPIKey(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&domain.APIKey{}).Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}
