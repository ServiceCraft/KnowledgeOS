package domain

type Company struct {
	BaseModel
	Name string `gorm:"type:text;not null" json:"name"`
	Tier string `gorm:"type:text;not null;default:local" json:"tier"`
}

// TableName executes the domain.Company.TableName operation.
func (Company) TableName() string { return "companies" }
