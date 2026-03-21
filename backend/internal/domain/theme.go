package domain

type Theme struct {
	SyncableModel
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
}

func (Theme) TableName() string { return "themes" }
