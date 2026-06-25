package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type KBEntityType string

const (
	KBEntityArticle KBEntityType = "article"
	KBEntityQA      KBEntityType = "qa"
	KBEntityPricing KBEntityType = "pricing"
)

type KBIndexOperation string

const (
	KBIndexOperationUpsert KBIndexOperation = "upsert"
	KBIndexOperationDelete KBIndexOperation = "delete"
)

type KBIndexJobStatus string

const (
	KBIndexJobPending    KBIndexJobStatus = "pending"
	KBIndexJobProcessing KBIndexJobStatus = "processing"
	KBIndexJobDone       KBIndexJobStatus = "done"
	KBIndexJobFailed     KBIndexJobStatus = "failed"
)

type KBEmbedding struct {
	BaseModel
	CompanyID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"company_id"`
	EntityType  KBEntityType `gorm:"type:text;not null" json:"entity_type"`
	EntityID    uuid.UUID    `gorm:"type:uuid;not null" json:"entity_id"`
	ChunkIdx    int          `gorm:"not null" json:"chunk_idx"`
	ThemeID     *uuid.UUID   `gorm:"type:uuid" json:"theme_id,omitempty"`
	Title       string       `gorm:"type:text;not null;default:''" json:"title"`
	Content     string       `gorm:"type:text;not null" json:"content"`
	ContentHash string       `gorm:"type:text;not null" json:"content_hash"`
	Model       string       `gorm:"type:text;not null" json:"model"`
	Dim         int          `gorm:"not null;default:256" json:"dim"`
}

func (KBEmbedding) TableName() string { return "kb_embeddings" }

type KBIndexJob struct {
	BaseModel
	CompanyID   uuid.UUID        `gorm:"type:uuid;not null;index" json:"company_id"`
	EntityType  KBEntityType     `gorm:"type:text;not null" json:"entity_type"`
	EntityID    uuid.UUID        `gorm:"type:uuid;not null" json:"entity_id"`
	Operation   KBIndexOperation `gorm:"type:text;not null" json:"operation"`
	Status      KBIndexJobStatus `gorm:"type:text;not null" json:"status"`
	Attempts    int              `gorm:"not null;default:0" json:"attempts"`
	LastError   *string          `gorm:"type:text" json:"last_error,omitempty"`
	AvailableAt time.Time        `json:"available_at"`
}

func (KBIndexJob) TableName() string { return "kb_index_jobs" }

type RAGChunk struct {
	CompanyID     uuid.UUID
	EntityType    KBEntityType
	EntityID      uuid.UUID
	ChunkIdx      int
	ThemeID       *uuid.UUID
	Title         string
	Content       string
	EmbeddingText string
	ContentHash   string
	Model         string
	Dim           int
	Embedding     []float32
}

type RAGCandidate struct {
	SourceID    string       `json:"source_id"`
	EntityType  KBEntityType `json:"entity_type"`
	EntityID    uuid.UUID    `json:"entity_id"`
	ChunkIdx    int          `json:"chunk_idx"`
	ThemeID     *uuid.UUID   `json:"theme_id,omitempty"`
	Title       string       `json:"title"`
	Content     string       `json:"content"`
	Snippet     string       `json:"snippet,omitempty"`
	Score       float64      `json:"score"`
	VectorRank  int          `json:"vector_rank,omitempty"`
	LexicalRank int          `json:"lexical_rank,omitempty"`
}

type RetrieveRequest struct {
	Query      string
	Types      []KBEntityType
	ThemeID    *uuid.UUID
	VectorTopK int
	HybridTopK int
	Rewrite    bool
}

type RetrieveResult struct {
	Query          string         `json:"query"`
	RewrittenQuery string         `json:"rewritten_query,omitempty"`
	Results        []RAGCandidate `json:"results"`
}

type KBEmbeddingRepository interface {
	UpsertChunks(ctx context.Context, companyID uuid.UUID, chunks []RAGChunk) error
	DeleteEntity(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID) error
	DeleteChunksExcept(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID, keepChunkIdx []int) error
	DeleteCompany(ctx context.Context, companyID uuid.UUID) error
	ListByEntity(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID) ([]KBEmbedding, error)
	VectorSearch(ctx context.Context, companyID uuid.UUID, embedding []float32, filter RetrieveRequest) ([]RAGCandidate, error)
	CountByCompany(ctx context.Context, companyID uuid.UUID) (int64, error)
}

type KBIndexJobRepository interface {
	Enqueue(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID, operation KBIndexOperation) error
	ClaimPending(ctx context.Context, limit int) ([]KBIndexJob, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, nextAttempt time.Time, maxAttempts int) error
	Stats(ctx context.Context, companyID uuid.UUID) (map[KBIndexJobStatus]int64, error)
}

type KnowledgeIndexScheduler interface {
	ScheduleUpsert(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID) error
	ScheduleDelete(ctx context.Context, companyID uuid.UUID, entityType KBEntityType, entityID uuid.UUID) error
}
