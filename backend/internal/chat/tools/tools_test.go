package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
)

func toolCall(name, args string) llm.ToolCall {
	return llm.ToolCall{ID: "call", Name: name, Arguments: json.RawMessage(args)}
}

type fakeRetriever struct {
	lastReq domain.RetrieveRequest
	results []domain.RAGCandidate
}

func (r *fakeRetriever) Search(_ context.Context, _ uuid.UUID, req domain.RetrieveRequest) (*domain.RetrieveResult, error) {
	r.lastReq = req
	return &domain.RetrieveResult{Query: req.Query, Results: r.results}, nil
}

func TestSearchKnowledgeToolReturnsSources(t *testing.T) {
	entityID := uuid.New()
	retriever := &fakeRetriever{results: []domain.RAGCandidate{{
		SourceID: "qa:" + entityID.String() + ":0", EntityType: domain.KBEntityQA, EntityID: entityID,
		Title: "Доставка", Content: "Бесплатно от 1000", Score: 0.7,
	}}}
	tool := NewSearchKnowledgeTool(retriever, 5)

	res, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{"query":"доставка","top_k":3}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if retriever.lastReq.HybridTopK != 3 || !retriever.lastReq.Rewrite {
		t.Fatalf("retrieve request = %+v", retriever.lastReq)
	}
	if len(res.Sources) != 1 || res.Sources[0].Title != "Доставка" {
		t.Fatalf("sources = %+v", res.Sources)
	}
	var payload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal([]byte(res.Content), &payload); err != nil {
		t.Fatalf("content not JSON: %v", err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("payload results = %+v", payload.Results)
	}
}

func TestSearchKnowledgeToolRequiresQuery(t *testing.T) {
	tool := NewSearchKnowledgeTool(&fakeRetriever{}, 5)
	_, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{"query":"   "}`))
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
}

type fakePricing struct {
	nodes []domain.PricingNode
}

func (p *fakePricing) List(_ context.Context, _ uuid.UUID, _ domain.PricingNodeFilter) ([]domain.PricingNode, int64, error) {
	return p.nodes, int64(len(p.nodes)), nil
}

func (p *fakePricing) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.PricingNode, error) {
	for i := range p.nodes {
		if p.nodes[i].ID == id {
			return &p.nodes[i], nil
		}
	}
	return nil, errors.New("not found")
}

func priceNode(name string, price float64) domain.PricingNode {
	node := domain.PricingNode{NodeType: "service", Name: name, Price: &price}
	node.ID = uuid.New()
	return node
}

func TestGetPricingToolFiltersByQuery(t *testing.T) {
	pricing := &fakePricing{nodes: []domain.PricingNode{
		priceNode("Стрижка", 1500),
		priceNode("Окрашивание", 4000),
	}}
	tool := NewGetPricingTool(pricing)

	res, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{"query":"стрижк"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(res.Sources) != 1 || res.Sources[0].Title != "Стрижка" {
		t.Fatalf("sources = %+v", res.Sources)
	}
	var payload struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal([]byte(res.Content), &payload); err != nil {
		t.Fatalf("content not JSON: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items = %+v", payload.Items)
	}
}

type fakeArticles struct{ article *domain.Article }

func (a *fakeArticles) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.Article, error) {
	if a.article == nil {
		return nil, errors.New("not found")
	}
	return a.article, nil
}

type fakeQA struct{ qa *domain.QAPair }

func (q *fakeQA) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.QAPair, error) {
	if q.qa == nil {
		return nil, errors.New("not found")
	}
	return q.qa, nil
}

func TestGetServiceInfoToolArticle(t *testing.T) {
	article := &domain.Article{Title: "Услуга", Body: "Описание услуги"}
	article.ID = uuid.New()
	tool := NewGetServiceInfoTool(&fakeArticles{article: article}, &fakeQA{}, &fakePricing{})

	args := `{"source_type":"article","source_id":"` + article.ID.String() + `"}`
	res, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(res.Sources) != 1 || res.Sources[0].EntityType != domain.KBEntityArticle {
		t.Fatalf("sources = %+v", res.Sources)
	}
}

func TestGetServiceInfoToolInvalidUUID(t *testing.T) {
	tool := NewGetServiceInfoTool(&fakeArticles{}, &fakeQA{}, &fakePricing{})
	_, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{"source_type":"article","source_id":"not-a-uuid"}`))
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
}
