package domain

import "github.com/google/uuid"

type Comment struct {
	SyncableModel
	EntityType string     `gorm:"type:text;not null" json:"entity_type"`
	EntityID   uuid.UUID  `gorm:"type:uuid;not null" json:"entity_id"`
	Body       string     `gorm:"type:text;not null" json:"body"`
	AuthorID   *uuid.UUID `gorm:"type:uuid" json:"author_id,omitempty"`
}

// TableName executes the domain.Comment.TableName operation.
func (Comment) TableName() string { return "comments" }
