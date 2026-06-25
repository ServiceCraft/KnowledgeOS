package domain

import (
	"time"

	"github.com/google/uuid"
)

type QAPair struct {
	SyncableModel
	ThemeID      *uuid.UUID `gorm:"type:uuid" json:"theme_id,omitempty"`
	Question     string     `gorm:"type:text;not null" json:"question"`
	Answer       string     `gorm:"type:text;not null" json:"answer"`
	IsFAQ        bool       `gorm:"not null;default:false" json:"is_faq"`
	IsLocked     bool       `gorm:"not null;default:false" json:"is_locked"`
	Frequency    int        `gorm:"not null;default:0" json:"frequency"`
	SearchVector string     `gorm:"type:tsvector;->" json:"-"`
	AIAnswer     *string    `gorm:"type:text" json:"ai_answer,omitempty"`
	AIStatus     *string    `gorm:"type:text" json:"ai_status,omitempty"`
	AIReviewedBy *uuid.UUID `gorm:"type:uuid" json:"ai_reviewed_by,omitempty"`
	AIReviewedAt *time.Time `json:"ai_reviewed_at,omitempty"`
}

// TableName executes the domain.QAPair.TableName operation.
func (QAPair) TableName() string { return "qa_pairs" }
