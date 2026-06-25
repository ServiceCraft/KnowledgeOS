package tools

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
)

const (
	defaultPricingLimit = 20
	maxPricingLimit     = 50
	pricingScanLimit    = 200
)

// pricingLister is the subset of the pricing service used by the get_pricing tool.
type pricingLister interface {
	List(ctx context.Context, companyID uuid.UUID, filter domain.PricingNodeFilter) ([]domain.PricingNode, int64, error)
}

// GetPricingTool exposes the price list to the model.
type GetPricingTool struct {
	pricing pricingLister
}

// NewGetPricingTool executes the tools.NewGetPricingTool operation.
func NewGetPricingTool(pricing pricingLister) *GetPricingTool {
	return &GetPricingTool{pricing: pricing}
}

// Name executes the tools.GetPricingTool.Name operation.
func (t *GetPricingTool) Name() string { return "get_pricing" }

// Description executes the tools.GetPricingTool.Description operation.
func (t *GetPricingTool) Description() string {
	return "Получить позиции прайс-листа компании. Можно отфильтровать по названию услуги."
}

// Schema executes the tools.GetPricingTool.Schema operation.
func (t *GetPricingTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Фильтр по названию позиции (необязательно)"},
			"limit": {"type": "integer", "minimum": 1, "maximum": 50, "description": "Максимум позиций в ответе"}
		}
	}`)
}

type getPricingArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type pricingItem struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	NodeType string   `json:"node_type"`
	Price    *float64 `json:"price,omitempty"`
}

// Execute executes the tools.GetPricingTool.Execute operation.
func (t *GetPricingTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetPricingTool.Execute")
	var in getPricingArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return Result{}, &ValidationError{Msg: "invalid arguments for get_pricing"}
		}
	}
	limit := in.Limit
	if limit <= 0 || limit > maxPricingLimit {
		limit = defaultPricingLimit
	}
	query := strings.ToLower(strings.TrimSpace(in.Query))

	fetch := limit
	if query != "" {
		fetch = pricingScanLimit
	}
	nodes, _, err := t.pricing.List(ctx, companyID, domain.PricingNodeFilter{Page: 1, Limit: fetch})
	if err != nil {
		return Result{}, err
	}

	items := make([]pricingItem, 0, limit)
	sources := make([]domain.ChatSource, 0, limit)
	for _, node := range nodes {
		if query != "" && !strings.Contains(strings.ToLower(node.Name), query) {
			continue
		}
		idStr := node.ID.String()
		items = append(items, pricingItem{
			ID:       idStr,
			Name:     node.Name,
			NodeType: node.NodeType,
			Price:    node.Price,
		})
		sources = append(sources, domain.ChatSource{
			SourceID:   entitySourceID(domain.KBEntityPricing, idStr),
			EntityType: domain.KBEntityPricing,
			EntityID:   node.ID,
			Title:      node.Name,
			Content:    pricingContent(node),
		})
		if len(items) >= limit {
			break
		}
	}

	payload, _ := json.Marshal(map[string]interface{}{"items": items})
	return Result{Content: string(payload), Sources: sources}, nil
}

func pricingContent(node domain.PricingNode) string {
	var b strings.Builder
	b.WriteString(node.Name)
	if node.Price != nil {
		b.WriteString(": ")
		b.WriteString(strconv.FormatFloat(*node.Price, 'f', -1, 64))
	}
	return b.String()
}
