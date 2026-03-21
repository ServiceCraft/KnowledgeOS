package domain

import "github.com/google/uuid"

type Role string

const (
	RoleSuperadmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleEditor     Role = "editor"
	RoleViewer     Role = "viewer"
)

type User struct {
	BaseModel
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"company_id,omitempty"`
	Email        string     `gorm:"type:text;not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Role         Role       `gorm:"type:text;not null;default:viewer" json:"role"`
}

func (User) TableName() string { return "users" }
