package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
)

const defaultSearchTopK = 8

// knowledgeRetriever is the subset of the retriever service used by the
// search_knowledge tool (Б2 hybrid retriever).
type knowledgeRetriever interface {
	Search(ctx context.Context, companyID uuid.UUID, req domain.RetrieveRequest) (*domain.RetrieveResult, error)
}

// SearchKnowledgeTool lets the model search the tenant knowledge base.
type SearchKnowledgeTool struct {
	retriever knowledgeRetriever
	maxTopK   int
}

// NewSearchKnowledgeTool executes the tools.NewSearchKnowledgeTool operation.
func NewSearchKnowledgeTool(retriever knowledgeRetriever, maxTopK int) *SearchKnowledgeTool {
	if maxTopK <= 0 {
		maxTopK = defaultSearchTopK
	}
	return &SearchKnowledgeTool{retriever: retriever, maxTopK: maxTopK}
}

// Name executes the tools.SearchKnowledgeTool.Name operation.
func (t *SearchKnowledgeTool) Name() string { return "search_knowledge" }

// Description executes the tools.SearchKnowledgeTool.Description operation.
func (t *SearchKnowledgeTool) Description() string {
	return "Найти информацию в базе знаний компании (статьи, вопросы-ответы, услуги) по текстовому запросу."
}

// Schema executes the tools.SearchKnowledgeTool.Schema operation.
func (t *SearchKnowledgeTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "minLength": 1, "description": "Поисковый запрос на естественном языке"},
			"top_k": {"type": "integer", "minimum": 1, "maximum": 10, "description": "Сколько фрагментов вернуть"}
		},
		"required": ["query"]
	}`)
}

type searchKnowledgeArgs struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type searchKnowledgeHit struct {
	SourceID string  `json:"source_id"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
}

// Execute executes the tools.SearchKnowledgeTool.Execute operation.
func (t *SearchKnowledgeTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.SearchKnowledgeTool.Execute")
	var in searchKnowledgeArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return Result{}, &ValidationError{Msg: "invalid arguments for search_knowledge"}
		}
	}
	in.Query = strings.TrimSpace(in.Query)
	if in.Query == "" {
		return Result{}, &ValidationError{Msg: "query is required"}
	}
	topK := in.TopK
	if topK <= 0 || topK > t.maxTopK {
		topK = t.maxTopK
	}

	res, err := t.retriever.Search(ctx, companyID, domain.RetrieveRequest{
		Query:      in.Query,
		HybridTopK: topK,
		Rewrite:    true,
	})
	if err != nil {
		return Result{}, err
	}

	sources := sourcesFromCandidates(res.Results, topK)
	hits := make([]searchKnowledgeHit, 0, len(sources))
	for _, src := range sources {
		hits = append(hits, searchKnowledgeHit{
			SourceID: src.SourceID,
			Title:    src.Title,
			Content:  src.Content,
			Score:    src.Score,
		})
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"query":   in.Query,
		"results": hits,
	})
	return Result{Content: string(payload), Sources: sources}, nil
}
