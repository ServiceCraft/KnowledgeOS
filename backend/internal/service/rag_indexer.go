package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	ragindexer "github.com/knowledgeos/backend/internal/rag/indexer"
)

type RAGIndexerConfig struct {
	WorkerEnabled      bool
	WorkerBatchSize    int
	WorkerPollInterval time.Duration
	IndexMaxAttempts   int
}

type RAGIndexerService struct {
	jobs       domain.KBIndexJobRepository
	embeddings domain.KBEmbeddingRepository
	articles   domain.ArticleRepository
	qa         domain.QAPairRepository
	pricing    domain.PricingNodeRepository
	llmFactory *LLMFactory
	chunker    *ragindexer.Chunker
	cfg        RAGIndexerConfig
}

// NewRAGIndexerService executes the service.NewRAGIndexerService operation.
func NewRAGIndexerService(
	jobs domain.KBIndexJobRepository,
	embeddings domain.KBEmbeddingRepository,
	articles domain.ArticleRepository,
	qa domain.QAPairRepository,
	pricing domain.PricingNodeRepository,
	llmFactory *LLMFactory,
	cfg RAGIndexerConfig,
) *RAGIndexerService {
	if cfg.WorkerBatchSize <= 0 {
		cfg.WorkerBatchSize = 10
	}
	if cfg.WorkerPollInterval <= 0 {
		cfg.WorkerPollInterval = 5 * time.Second
	}
	if cfg.IndexMaxAttempts <= 0 {
		cfg.IndexMaxAttempts = 5
	}
	return &RAGIndexerService{
		jobs:       jobs,
		embeddings: embeddings,
		articles:   articles,
		qa:         qa,
		pricing:    pricing,
		llmFactory: llmFactory,
		chunker:    ragindexer.NewChunker(),
		cfg:        cfg,
	}
}

// ScheduleUpsert executes the service.RAGIndexerService.ScheduleUpsert operation.
func (s *RAGIndexerService) ScheduleUpsert(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) error {
	applog.TraceCall(ctx, "service.RAGIndexerService.ScheduleUpsert")
	if s == nil || s.jobs == nil {
		return nil
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("entity_type", string(entityType)).
		Str("entity_id", entityID.String()).
		Msg("rag index upsert scheduled")
	return s.jobs.Enqueue(ctx, companyID, entityType, entityID, domain.KBIndexOperationUpsert)
}

// ScheduleDelete executes the service.RAGIndexerService.ScheduleDelete operation.
func (s *RAGIndexerService) ScheduleDelete(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) error {
	applog.TraceCall(ctx, "service.RAGIndexerService.ScheduleDelete")
	if s == nil || s.jobs == nil {
		return nil
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("entity_type", string(entityType)).
		Str("entity_id", entityID.String()).
		Msg("rag index delete scheduled")
	return s.jobs.Enqueue(ctx, companyID, entityType, entityID, domain.KBIndexOperationDelete)
}

// StartWorker executes the service.RAGIndexerService.StartWorker operation.
func (s *RAGIndexerService) StartWorker(ctx context.Context) {
	applog.TraceCall(ctx, "service.RAGIndexerService.StartWorker")
	if s == nil || !s.cfg.WorkerEnabled {
		return
	}
	applog.From(ctx).Info().
		Int("batch_size", s.cfg.WorkerBatchSize).
		Dur("poll_interval", s.cfg.WorkerPollInterval).
		Int("max_attempts", s.cfg.IndexMaxAttempts).
		Msg("rag index worker starting")
	go s.worker(ctx)
}

func (s *RAGIndexerService) worker(ctx context.Context) {
	applog.TraceCall(ctx, "service.RAGIndexerService.worker")
	ticker := time.NewTicker(s.cfg.WorkerPollInterval)
	defer ticker.Stop()
	for {
		s.processPending(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *RAGIndexerService) processPending(ctx context.Context) {
	applog.TraceCall(ctx, "service.RAGIndexerService.processPending")
	jobs, err := s.jobs.ClaimPending(ctx, s.cfg.WorkerBatchSize)
	if err != nil {
		applog.From(ctx).Error().Err(err).Msg("rag indexer claim failed")
		return
	}
	for _, job := range jobs {
		if err := s.ProcessJob(ctx, job); err != nil {
			next := time.Now().Add(backoff(job.Attempts))
			if markErr := s.jobs.MarkFailed(ctx, job.ID, truncateError(err), next, s.cfg.IndexMaxAttempts); markErr != nil {
				applog.From(ctx).Error().Err(markErr).Str("job_id", job.ID.String()).Msg("rag indexer mark failed")
			}
			continue
		}
		if err := s.jobs.MarkDone(ctx, job.ID); err != nil {
			applog.From(ctx).Error().Err(err).Str("job_id", job.ID.String()).Msg("rag indexer mark done failed")
		}
	}
}

// ProcessJob executes the service.RAGIndexerService.ProcessJob operation.
func (s *RAGIndexerService) ProcessJob(ctx context.Context, job domain.KBIndexJob) error {
	applog.TraceCall(ctx, "service.RAGIndexerService.ProcessJob")
	log := applog.From(ctx).With().
		Str("job_id", job.ID.String()).
		Str("company_id", job.CompanyID.String()).
		Str("entity_type", string(job.EntityType)).
		Str("entity_id", job.EntityID.String()).
		Str("operation", string(job.Operation)).
		Int("attempts", job.Attempts).
		Logger()
	log.Debug().Msg("rag index job processing")
	if job.Operation == domain.KBIndexOperationDelete {
		err := s.embeddings.DeleteEntity(ctx, job.CompanyID, job.EntityType, job.EntityID)
		if err != nil {
			log.Error().Err(err).Msg("rag index delete failed")
			return err
		}
		log.Debug().Msg("rag index delete completed")
		return nil
	}
	chunks, err := s.buildChunks(ctx, job.CompanyID, job.EntityType, job.EntityID)
	if err != nil {
		if errors.Is(err, errIndexEntityNotFound) {
			log.Debug().Msg("rag index entity missing, deleting embeddings")
			return s.embeddings.DeleteEntity(ctx, job.CompanyID, job.EntityType, job.EntityID)
		}
		log.Error().Err(err).Msg("rag index build chunks failed")
		return err
	}
	if err := s.embedAndUpsert(ctx, job.CompanyID, job.EntityType, job.EntityID, chunks); err != nil {
		log.Error().Err(err).Int("chunks", len(chunks)).Msg("rag index upsert failed")
		return err
	}
	log.Debug().Int("chunks", len(chunks)).Msg("rag index job completed")
	return nil
}

// ReindexCompany executes the service.RAGIndexerService.ReindexCompany operation.
func (s *RAGIndexerService) ReindexCompany(ctx context.Context, companyID uuid.UUID) error {
	applog.TraceCall(ctx, "service.RAGIndexerService.ReindexCompany")
	applog.From(ctx).Info().Str("company_id", companyID.String()).Msg("rag company reindex requested")
	if err := s.embeddings.DeleteCompany(ctx, companyID); err != nil {
		return err
	}
	articles, _, err := s.articles.List(ctx, companyID, domain.ArticleFilter{Limit: 10000})
	if err != nil {
		return err
	}
	for _, article := range articles {
		if err := s.ScheduleUpsert(ctx, companyID, domain.KBEntityArticle, article.ID); err != nil {
			return err
		}
	}
	qaPairs, _, err := s.qa.List(ctx, companyID, domain.QAPairFilter{Limit: 10000})
	if err != nil {
		return err
	}
	for _, qa := range qaPairs {
		if err := s.ScheduleUpsert(ctx, companyID, domain.KBEntityQA, qa.ID); err != nil {
			return err
		}
	}
	nodes, _, err := s.pricing.List(ctx, companyID, domain.PricingNodeFilter{Limit: 10000})
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if err := s.ScheduleUpsert(ctx, companyID, domain.KBEntityPricing, node.ID); err != nil {
			return err
		}
	}
	return nil
}

// IndexStatus executes the service.RAGIndexerService.IndexStatus operation.
func (s *RAGIndexerService) IndexStatus(ctx context.Context, companyID uuid.UUID) (map[string]interface{}, error) {
	applog.TraceCall(ctx, "service.RAGIndexerService.IndexStatus")
	total, err := s.embeddings.CountByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	stats, err := s.jobs.Stats(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"embeddings": total,
		"jobs":       stats,
	}, nil
}

var errIndexEntityNotFound = errors.New("index entity not found")

func (s *RAGIndexerService) buildChunks(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) ([]domain.RAGChunk, error) {
	applog.TraceCall(ctx, "service.RAGIndexerService.buildChunks")
	switch entityType {
	case domain.KBEntityArticle:
		article, err := s.articles.GetByID(ctx, companyID, entityID)
		if err != nil {
			return nil, errIndexEntityNotFound
		}
		return s.chunker.ChunksForArticle(companyID, article), nil
	case domain.KBEntityQA:
		qa, err := s.qa.GetByID(ctx, companyID, entityID)
		if err != nil {
			return nil, errIndexEntityNotFound
		}
		return s.chunker.ChunksForQA(companyID, qa), nil
	case domain.KBEntityPricing:
		node, err := s.pricing.GetByID(ctx, companyID, entityID)
		if err != nil {
			return nil, errIndexEntityNotFound
		}
		return s.chunker.ChunksForPricing(companyID, node, s.pricingPath(ctx, companyID, node)), nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

func (s *RAGIndexerService) pricingPath(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) []string {
	applog.TraceCall(ctx, "service.RAGIndexerService.pricingPath")
	var path []string
	parentID := node.ParentID
	for i := 0; parentID != nil && i < 16; i++ {
		parent, err := s.pricing.GetByID(ctx, companyID, *parentID)
		if err != nil {
			break
		}
		path = append([]string{parent.Name}, path...)
		parentID = parent.ParentID
	}
	return path
}

func (s *RAGIndexerService) embedAndUpsert(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID, chunks []domain.RAGChunk) error {
	applog.TraceCall(ctx, "service.RAGIndexerService.embedAndUpsert")
	if len(chunks) == 0 {
		applog.From(ctx).Debug().
			Str("company_id", companyID.String()).
			Str("entity_type", string(entityType)).
			Str("entity_id", entityID.String()).
			Msg("rag index has no chunks, deleting entity embeddings")
		return s.embeddings.DeleteEntity(ctx, companyID, entityType, entityID)
	}
	existing, err := s.embeddings.ListByEntity(ctx, companyID, entityType, entityID)
	if err != nil {
		return err
	}
	existingHash := map[int]string{}
	for _, item := range existing {
		existingHash[item.ChunkIdx] = item.ContentHash
	}

	keep := make([]int, 0, len(chunks))
	var changed []domain.RAGChunk
	var texts []string
	for _, chunk := range chunks {
		keep = append(keep, chunk.ChunkIdx)
		if existingHash[chunk.ChunkIdx] == chunk.ContentHash {
			continue
		}
		changed = append(changed, chunk)
		texts = append(texts, chunk.EmbeddingText)
	}
	if err := s.embeddings.DeleteChunksExcept(ctx, companyID, entityType, entityID, keep); err != nil {
		return err
	}
	if len(changed) == 0 {
		applog.From(ctx).Debug().
			Str("company_id", companyID.String()).
			Str("entity_type", string(entityType)).
			Str("entity_id", entityID.String()).
			Int("chunks", len(chunks)).
			Msg("rag index chunks unchanged")
		return nil
	}
	_, embedder, err := s.llmFactory.ForCompany(ctx, companyID)
	if err != nil {
		return err
	}
	vectors, err := embedder.EmbedDocs(ctx, texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(changed) {
		return fmt.Errorf("embedding count mismatch: got %d want %d", len(vectors), len(changed))
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("entity_type", string(entityType)).
		Str("entity_id", entityID.String()).
		Int("changed_chunks", len(changed)).
		Int("total_chunks", len(chunks)).
		Msg("rag index embeddings generated")
	model := s.llmFactory.embeddingModel(s.llmFactory.cfg.YandexEmbeddingDocModel, "text-embeddings-v2-doc")
	for i := range changed {
		changed[i].Embedding = vectors[i]
		changed[i].Dim = len(vectors[i])
		changed[i].Model = model
	}
	return s.embeddings.UpsertChunks(ctx, companyID, changed)
}

func backoff(attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<attempts) * time.Second
}

func truncateError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 1000 {
		return msg[:1000]
	}
	return msg
}
