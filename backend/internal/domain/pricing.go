package domain

import "github.com/google/uuid"

type PricingNode struct {
	SyncableModel
	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	NodeType string     `gorm:"type:text;not null" json:"node_type"`
	Name     string     `gorm:"type:text;not null" json:"name"`
	Price    *float64   `gorm:"type:numeric(12,2)" json:"price,omitempty"`
}

func (PricingNode) TableName() string { return "pricing_nodes" }
