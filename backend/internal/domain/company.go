package domain

type Company struct {
	BaseModel
	Name string `gorm:"type:text;not null" json:"name"`
	Tier string `gorm:"type:text;not null;default:local" json:"tier"`
}

func (Company) TableName() string { return "companies" }
