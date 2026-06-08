package domain

import "github.com/google/uuid"

type Role string

const (
	RoleSuperadmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleEditor     Role = "editor"
	RoleViewer     Role = "viewer"
)

// ValidRole reports whether r is one of the known platform roles.
func ValidRole(r Role) bool {
	switch r {
	case RoleSuperadmin, RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

type User struct {
	BaseModel
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"company_id,omitempty"`
	Email        string     `gorm:"type:text;not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Role         Role       `gorm:"type:text;not null;default:viewer" json:"role"`
	// No GORM default tag: with one, GORM omits a false value on INSERT (DB
	// default would force true), so creating an inactive user would silently be
	// stored as active. The column default lives in the migration; every Go
	// create path sets IsActive explicitly.
	IsActive bool `gorm:"not null" json:"is_active"`
}

func (User) TableName() string { return "users" }
