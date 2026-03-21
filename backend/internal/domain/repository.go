package domain

import (
	"context"

	"github.com/google/uuid"
)

// Filters

type QAPairFilter struct {
	ThemeID *uuid.UUID
	IsFAQ   *bool
	Page    int
	Limit   int
	Query   string
}

type ThemeFilter struct {
	Page  int
	Limit int
	Query string
}

type PricingNodeFilter struct {
	ParentID *uuid.UUID
	NodeType string
	Page     int
	Limit    int
}

type ArticleFilter struct {
	Page  int
	Limit int
	Query string
}

type CommentFilter struct {
	Page int
	Limit int
}

type EntityLinkFilter struct {
	Page int
	Limit int
}

type CompanyFilter struct {
	Page  int
	Limit int
}

type SearchFilter struct {
	Query   string
	Types   []string
	ThemeID *uuid.UUID
	Page    int
	Limit   int
}

type SearchResult struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Title      string    `json:"title"`
	Snippet    string    `json:"snippet"`
	Rank       float64   `json:"rank"`
}

// Repositories

type QAPairRepository interface {
	List(ctx context.Context, companyID uuid.UUID, filter QAPairFilter) ([]QAPair, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*QAPair, error)
	Create(ctx context.Context, companyID uuid.UUID, qa *QAPair) error
	Update(ctx context.Context, companyID uuid.UUID, qa *QAPair) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]QAPair, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, qa *QAPair) error
}

type ThemeRepository interface {
	List(ctx context.Context, companyID uuid.UUID, filter ThemeFilter) ([]Theme, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*Theme, error)
	Create(ctx context.Context, companyID uuid.UUID, theme *Theme) error
	Update(ctx context.Context, companyID uuid.UUID, theme *Theme) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]Theme, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, theme *Theme) error
}

type PricingNodeRepository interface {
	List(ctx context.Context, companyID uuid.UUID, filter PricingNodeFilter) ([]PricingNode, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*PricingNode, error)
	Create(ctx context.Context, companyID uuid.UUID, node *PricingNode) error
	Update(ctx context.Context, companyID uuid.UUID, node *PricingNode) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]PricingNode, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, node *PricingNode) error
}

type ArticleRepository interface {
	List(ctx context.Context, companyID uuid.UUID, filter ArticleFilter) ([]Article, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*Article, error)
	Create(ctx context.Context, companyID uuid.UUID, article *Article) error
	Update(ctx context.Context, companyID uuid.UUID, article *Article) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]Article, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, article *Article) error
}

type CommentRepository interface {
	List(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID, filter CommentFilter) ([]Comment, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*Comment, error)
	Create(ctx context.Context, companyID uuid.UUID, comment *Comment) error
	Update(ctx context.Context, companyID uuid.UUID, comment *Comment) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]Comment, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, comment *Comment) error
}

type EntityLinkRepository interface {
	List(ctx context.Context, companyID uuid.UUID, sourceType string, sourceID uuid.UUID, filter EntityLinkFilter) ([]EntityLink, int64, error)
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*EntityLink, error)
	Create(ctx context.Context, companyID uuid.UUID, link *EntityLink) error
	Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error
	ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]EntityLink, error)
	ApplyRemote(ctx context.Context, companyID uuid.UUID, link *EntityLink) error
}

type CompanyRepository interface {
	List(ctx context.Context, filter CompanyFilter) ([]Company, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	Create(ctx context.Context, company *Company) error
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

type SearchRepository interface {
	Search(ctx context.Context, companyID uuid.UUID, filter SearchFilter) ([]SearchResult, int64, error)
}

type SyncRepository interface {
	GetSequence(ctx context.Context, companyID uuid.UUID) (*SyncSequence, error)
	InitSequence(ctx context.Context, companyID uuid.UUID) error
	GetWatermark(ctx context.Context, companyID uuid.UUID) (*SyncWatermark, error)
	UpdateWatermark(ctx context.Context, wm *SyncWatermark) error
	GetStatus(ctx context.Context, companyID uuid.UUID) (*SyncStatus, error)
	UpdateStatus(ctx context.Context, status *SyncStatus) error
	CreateLog(ctx context.Context, entry *SyncLog) error

	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	CreateAPIKey(ctx context.Context, key *APIKey) error
	TouchAPIKey(ctx context.Context, id uuid.UUID) error
}
