package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
)

// articleGetter, qaGetter and pricingGetter are the subsets of the knowledge
// base services used by the get_service_info tool.
type articleGetter interface {
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Article, error)
}

type qaGetter interface {
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.QAPair, error)
}

type pricingGetter interface {
	GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.PricingNode, error)
}

// GetServiceInfoTool fetches a specific knowledge base entry by type and id,
// typically a source returned earlier by search_knowledge.
type GetServiceInfoTool struct {
	articles articleGetter
	qa       qaGetter
	pricing  pricingGetter
}

// NewGetServiceInfoTool executes the tools.NewGetServiceInfoTool operation.
func NewGetServiceInfoTool(articles articleGetter, qa qaGetter, pricing pricingGetter) *GetServiceInfoTool {
	return &GetServiceInfoTool{articles: articles, qa: qa, pricing: pricing}
}

// Name executes the tools.GetServiceInfoTool.Name operation.
func (t *GetServiceInfoTool) Name() string { return "get_service_info" }

// Description executes the tools.GetServiceInfoTool.Description operation.
func (t *GetServiceInfoTool) Description() string {
	return "Получить полную информацию об услуге или материале базы знаний по типу и идентификатору источника."
}

// Schema executes the tools.GetServiceInfoTool.Schema operation.
func (t *GetServiceInfoTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"source_type": {"type": "string", "enum": ["article", "qa", "pricing"], "description": "Тип источника"},
			"source_id": {"type": "string", "minLength": 1, "description": "UUID источника"}
		},
		"required": ["source_type", "source_id"]
	}`)
}

type getServiceInfoArgs struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

// Execute executes the tools.GetServiceInfoTool.Execute operation.
func (t *GetServiceInfoTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetServiceInfoTool.Execute")
	var in getServiceInfoArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return Result{}, &ValidationError{Msg: "invalid arguments for get_service_info"}
		}
	}
	in.SourceType = strings.TrimSpace(in.SourceType)
	entityID, err := uuid.Parse(strings.TrimSpace(in.SourceID))
	if err != nil {
		return Result{}, &ValidationError{Msg: "source_id must be a valid UUID"}
	}

	switch domain.KBEntityType(in.SourceType) {
	case domain.KBEntityArticle:
		article, err := t.articles.GetByID(ctx, companyID, entityID)
		if err != nil {
			return Result{}, fmt.Errorf("article not found")
		}
		return buildServiceInfoResult(domain.KBEntityArticle, entityID, article.Title, article.Body), nil
	case domain.KBEntityQA:
		qa, err := t.qa.GetByID(ctx, companyID, entityID)
		if err != nil {
			return Result{}, fmt.Errorf("qa pair not found")
		}
		content := "Вопрос: " + qa.Question + "\nОтвет: " + qa.Answer
		return buildServiceInfoResult(domain.KBEntityQA, entityID, qa.Question, content), nil
	case domain.KBEntityPricing:
		node, err := t.pricing.GetByID(ctx, companyID, entityID)
		if err != nil {
			return Result{}, fmt.Errorf("pricing node not found")
		}
		return buildServiceInfoResult(domain.KBEntityPricing, entityID, node.Name, pricingContent(*node)), nil
	default:
		return Result{}, &ValidationError{Msg: "source_type must be one of: article, qa, pricing"}
	}
}

func buildServiceInfoResult(entityType domain.KBEntityType, entityID uuid.UUID, title, content string) Result {
	sourceID := entitySourceID(entityType, entityID.String())
	source := domain.ChatSource{
		SourceID:   sourceID,
		EntityType: entityType,
		EntityID:   entityID,
		Title:      title,
		Content:    content,
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"source_id":   sourceID,
		"entity_type": entityType,
		"title":       title,
		"content":     content,
	})
	return Result{Content: string(payload), Sources: []domain.ChatSource{source}}
}
