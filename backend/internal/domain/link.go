package domain

import "github.com/google/uuid"

type EntityLink struct {
	SyncableModel
	SourceType string     `gorm:"type:text;not null" json:"source_type"`
	SourceID   uuid.UUID  `gorm:"type:uuid;not null" json:"source_id"`
	TargetType *string    `gorm:"type:text" json:"target_type,omitempty"`
	TargetID   *uuid.UUID `gorm:"type:uuid" json:"target_id,omitempty"`
	URL        *string    `gorm:"type:text" json:"url,omitempty"`
	Label      *string    `gorm:"type:text" json:"label,omitempty"`
}

func (EntityLink) TableName() string { return "entity_links" }
