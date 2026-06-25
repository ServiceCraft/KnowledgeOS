package service

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
	applog "github.com/knowledgeos/backend/internal/logger"
)

const rrfK = 60.0

type RetrieverService struct {
	embeddings domain.KBEmbeddingRepository
	search     domain.SearchRepository
	llmFactory *LLMFactory
	vectorTopK int
	hybridTopK int
}

func NewRetrieverService(embeddings domain.KBEmbeddingRepository, search domain.SearchRepository, llmFactory *LLMFactory, vectorTopK, hybridTopK int) *RetrieverService {
	if vectorTopK <= 0 {
		vectorTopK = 20
	}
	if hybridTopK <= 0 {
		hybridTopK = 8
	}
	return &RetrieverService{embeddings: embeddings, search: search, llmFactory: llmFactory, vectorTopK: vectorTopK, hybridTopK: hybridTopK}
}

func (s *RetrieverService) Search(ctx context.Context, companyID uuid.UUID, req domain.RetrieveRequest) (*domain.RetrieveResult, error) {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return nil, badRequest("query is required")
	}
	log := applog.From(ctx).With().
		Str("company_id", companyID.String()).
		Int("types", len(req.Types)).
		Bool("rewrite", req.Rewrite).
		Logger()
	log.Debug().Msg("retriever search started")
	if req.VectorTopK <= 0 {
		req.VectorTopK = s.vectorTopK
	}
	if req.HybridTopK <= 0 {
		req.HybridTopK = s.hybridTopK
	}

	var provider llm.Provider
	var embedder llm.Embedder
	var llmErr error = badRequest("llm factory is not configured")
	if s.llmFactory != nil {
		provider, embedder, llmErr = s.llmFactory.ForCompany(ctx, companyID)
	}
	if llmErr != nil {
		log.Debug().Err(llmErr).Msg("retriever vector branch disabled")
	}
	searchQuery := req.Query
	rewritten := ""
	if llmErr == nil && req.Rewrite {
		if candidate := s.rewriteQuery(ctx, provider, req.Query); candidate != "" {
			rewritten = candidate
			searchQuery = candidate
			log.Debug().Msg("retriever query rewritten")
		}
	}

	var vector []domain.RAGCandidate
	if llmErr == nil {
		if queryEmbedding, err := embedder.EmbedQuery(ctx, searchQuery); err == nil && len(queryEmbedding) > 0 {
			vectorReq := req
			vectorReq.Query = searchQuery
			vector, err = s.embeddings.VectorSearch(ctx, companyID, queryEmbedding, vectorReq)
			if err != nil {
				log.Error().Err(err).Msg("retriever vector search failed")
				return nil, err
			}
		} else if err != nil {
			log.Debug().Err(err).Msg("retriever query embedding failed")
		}
	}

	lexical, err := s.lexicalSearch(ctx, companyID, searchQuery, req)
	if err != nil {
		log.Error().Err(err).Msg("retriever lexical search failed")
		return nil, err
	}

	results := mergeRRF(vector, lexical, req.HybridTopK)
	log.Debug().
		Int("vector_results", len(vector)).
		Int("lexical_results", len(lexical)).
		Int("results", len(results)).
		Msg("retriever search completed")
	return &domain.RetrieveResult{
		Query:          req.Query,
		RewrittenQuery: rewritten,
		Results:        results,
	}, nil
}

func (s *RetrieverService) rewriteQuery(ctx context.Context, provider llm.Provider, query string) string {
	resp, err := provider.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Переформулируй реплику пользователя в короткий самостоятельный поисковый запрос для базы знаний. Верни только запрос без пояснений."},
			{Role: llm.RoleUser, Content: query},
		},
		GenerationParams: llm.GenerationParams{Temperature: 0, MaxTokens: 64},
	})
	if err != nil || resp == nil {
		return ""
	}
	out := strings.TrimSpace(resp.Message.Content)
	if out == "" || len([]rune(out)) > 300 {
		return ""
	}
	return out
}

func (s *RetrieverService) lexicalSearch(ctx context.Context, companyID uuid.UUID, query string, req domain.RetrieveRequest) ([]domain.RAGCandidate, error) {
	types := make([]string, 0, len(req.Types))
	for _, t := range req.Types {
		types = append(types, string(t))
	}
	items, _, err := s.search.Search(ctx, companyID, domain.SearchFilter{
		Query:   query,
		Types:   types,
		ThemeID: req.ThemeID,
		Limit:   req.VectorTopK,
		Page:    1,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.RAGCandidate, 0, len(items))
	for i, item := range items {
		entityType := domain.KBEntityType(item.EntityType)
		out = append(out, domain.RAGCandidate{
			SourceID:    string(entityType) + ":" + item.EntityID.String() + ":0",
			EntityType:  entityType,
			EntityID:    item.EntityID,
			ChunkIdx:    0,
			Title:       item.Title,
			Content:     item.Snippet,
			Snippet:     item.Snippet,
			Score:       item.Rank,
			LexicalRank: i + 1,
		})
	}
	return out, nil
}

func mergeRRF(vector, lexical []domain.RAGCandidate, limit int) []domain.RAGCandidate {
	if limit <= 0 {
		limit = 8
	}
	type acc struct {
		item  domain.RAGCandidate
		score float64
	}
	byKey := map[string]*acc{}
	add := func(items []domain.RAGCandidate, isVector bool) {
		for i, item := range items {
			rank := i + 1
			key := item.SourceID
			if key == "" {
				key = string(item.EntityType) + ":" + item.EntityID.String() + ":" + strconv.Itoa(item.ChunkIdx)
			}
			current := byKey[key]
			if current == nil {
				copy := item
				current = &acc{item: copy}
				byKey[key] = current
			}
			current.score += 1.0 / (rrfK + float64(rank))
			if isVector {
				current.item.VectorRank = rank
			} else {
				current.item.LexicalRank = rank
				if current.item.Content == "" {
					current.item.Content = item.Content
				}
				if current.item.Snippet == "" {
					current.item.Snippet = item.Snippet
				}
			}
		}
	}
	add(vector, true)
	add(lexical, false)

	out := make([]domain.RAGCandidate, 0, len(byKey))
	for _, current := range byKey {
		current.item.Score = current.score
		out = append(out, current.item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Title < out[j].Title
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
