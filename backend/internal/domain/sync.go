package domain

import (
	"time"

	"github.com/google/uuid"
)

type SyncSequence struct {
	CompanyID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"company_id"`
	CurrentSeq int64     `gorm:"not null;default:0" json:"current_seq"`
}

// TableName executes the domain.SyncSequence.TableName operation.
func (SyncSequence) TableName() string { return "sync_sequence" }

type SyncWatermark struct {
	CompanyID    uuid.UUID  `gorm:"type:uuid;primaryKey" json:"company_id"`
	LastLocalSeq int64      `gorm:"not null;default:0" json:"last_local_seq"`
	LastCloudSeq int64      `gorm:"not null;default:0" json:"last_cloud_seq"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
}

// TableName executes the domain.SyncWatermark.TableName operation.
func (SyncWatermark) TableName() string { return "sync_watermarks" }

type SyncStatus struct {
	CompanyID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"company_id"`
	LastSyncAt         *time.Time `json:"last_sync_at,omitempty"`
	LastSyncResult     *string    `gorm:"type:text" json:"last_sync_result,omitempty"`
	LastError          *string    `gorm:"type:text" json:"last_error,omitempty"`
	SubscriptionActive bool       `gorm:"not null;default:false" json:"subscription_active"`
}

// TableName executes the domain.SyncStatus.TableName operation.
func (SyncStatus) TableName() string { return "sync_status" }

type SyncLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID    uuid.UUID `gorm:"type:uuid;not null" json:"company_id"`
	Direction    string    `gorm:"type:text;not null" json:"direction"`
	EntityType   string    `gorm:"type:text;not null" json:"entity_type"`
	EntityID     uuid.UUID `gorm:"type:uuid;not null" json:"entity_id"`
	Seq          int64     `gorm:"not null" json:"seq"`
	Status       string    `gorm:"type:text;not null" json:"status"`
	ConflictNote *string   `gorm:"type:text" json:"conflict_note,omitempty"`
	SyncedAt     time.Time `gorm:"not null;default:now()" json:"synced_at"`
}

// TableName executes the domain.SyncLog.TableName operation.
func (SyncLog) TableName() string { return "sync_log" }
