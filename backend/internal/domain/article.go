package domain

type Article struct {
	SyncableModel
	Title        string `gorm:"type:text;not null" json:"title"`
	Body         string `gorm:"type:text;not null;default:''" json:"body"`
	SearchVector string `gorm:"type:tsvector;->" json:"-"`
}

func (Article) TableName() string { return "articles" }
