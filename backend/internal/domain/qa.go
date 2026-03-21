package domain

import "github.com/google/uuid"

type QAPair struct {
	SyncableModel
	ThemeID      *uuid.UUID `gorm:"type:uuid" json:"theme_id,omitempty"`
	Question     string     `gorm:"type:text;not null" json:"question"`
	Answer       string     `gorm:"type:text;not null" json:"answer"`
	IsFAQ        bool       `gorm:"not null;default:false" json:"is_faq"`
	IsLocked     bool       `gorm:"not null;default:false" json:"is_locked"`
	SearchVector string     `gorm:"type:tsvector;->" json:"-"`
}

func (QAPair) TableName() string { return "qa_pairs" }
