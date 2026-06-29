package domain

import "github.com/google/uuid"

// UserCompany links a user to a tenant company they may access.
type UserCompany struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	CompanyID uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
}

// TableName executes the domain.UserCompany.TableName operation.
func (UserCompany) TableName() string { return "user_companies" }
