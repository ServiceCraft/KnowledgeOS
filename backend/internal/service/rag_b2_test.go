package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

func TestMergeRRFCombinesRanks(t *testing.T) {
	entityID := uuid.New()
	vector := []domain.RAGCandidate{
		{SourceID: "qa:" + entityID.String() + ":0", EntityType: domain.KBEntityQA, EntityID: entityID, ChunkIdx: 0, Title: "vector"},
	}
	lexical := []domain.RAGCandidate{
		{SourceID: "qa:" + entityID.String() + ":0", EntityType: domain.KBEntityQA, EntityID: entityID, ChunkIdx: 0, Title: "lexical"},
	}

	results := mergeRRF(vector, lexical, 8)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].VectorRank != 1 || results[0].LexicalRank != 1 {
		t.Fatalf("ranks not merged: %+v", results[0])
	}
	if results[0].Score <= 1.0/rrfK {
		t.Fatalf("score was not accumulated: %f", results[0].Score)
	}
}

func TestRetrieverFallsBackToLexicalWithoutLLMFactory(t *testing.T) {
	companyID := uuid.New()
	entityID := uuid.New()
	search := &fakeSearchRepo{
		results: []domain.SearchResult{{
			EntityType: "pricing",
			EntityID:   entityID,
			Title:      "Стерилизация кошки",
			Snippet:    "Стерилизация кошки — 3500",
			Rank:       0.9,
		}},
	}
	svc := NewRetrieverService(nil, search, nil, 20, 8)

	result, err := svc.Search(context.Background(), companyID, domain.RetrieveRequest{Query: "сколько стерилизовать кошку", Rewrite: true})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Query != "сколько стерилизовать кошку" || result.RewrittenQuery != "" {
		t.Fatalf("unexpected query fields: %+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].EntityType != domain.KBEntityPricing {
		t.Fatalf("unexpected results: %+v", result.Results)
	}
	if search.companyID != companyID {
		t.Fatalf("companyID = %s, want %s", search.companyID, companyID)
	}
}

type fakeSearchRepo struct {
	companyID uuid.UUID
	filter    domain.SearchFilter
	results   []domain.SearchResult
}

func (r *fakeSearchRepo) Search(_ context.Context, companyID uuid.UUID, filter domain.SearchFilter) ([]domain.SearchResult, int64, error) {
	r.companyID = companyID
	r.filter = filter
	return r.results, int64(len(r.results)), nil
}

func TestArticleServiceSchedulesIndexJobs(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	articleID := uuid.New()
	articles := &fakeArticleRepo{items: map[uuid.UUID]*domain.Article{}}
	scheduler := &fakeIndexScheduler{}
	svc := NewArticleService(articles, scheduler)

	article := &domain.Article{Title: "Заголовок", Body: "Текст"}
	if err := svc.Create(ctx, companyID, article); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if scheduler.lastOp != domain.KBIndexOperationUpsert || scheduler.lastType != domain.KBEntityArticle {
		t.Fatalf("unexpected schedule after create: %+v", scheduler)
	}

	article.ID = articleID
	articles.items[articleID] = article
	if err := svc.Delete(ctx, companyID, articleID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if scheduler.lastOp != domain.KBIndexOperationDelete || scheduler.lastID != articleID {
		t.Fatalf("unexpected schedule after delete: %+v", scheduler)
	}
}

type fakeIndexScheduler struct {
	lastCompany uuid.UUID
	lastType    domain.KBEntityType
	lastID      uuid.UUID
	lastOp      domain.KBIndexOperation
}

func (s *fakeIndexScheduler) ScheduleUpsert(_ context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) error {
	s.lastCompany = companyID
	s.lastType = entityType
	s.lastID = entityID
	s.lastOp = domain.KBIndexOperationUpsert
	return nil
}

func (s *fakeIndexScheduler) ScheduleDelete(_ context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) error {
	s.lastCompany = companyID
	s.lastType = entityType
	s.lastID = entityID
	s.lastOp = domain.KBIndexOperationDelete
	return nil
}

type fakeArticleRepo struct {
	items map[uuid.UUID]*domain.Article
}

func (r *fakeArticleRepo) List(_ context.Context, _ uuid.UUID, _ domain.ArticleFilter) ([]domain.Article, int64, error) {
	out := make([]domain.Article, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, *item)
	}
	return out, int64(len(out)), nil
}

func (r *fakeArticleRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.Article, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeArticleRepo) Create(_ context.Context, companyID uuid.UUID, article *domain.Article) error {
	if article.ID == uuid.Nil {
		article.ID = uuid.New()
	}
	article.CompanyID = companyID
	copy := *article
	r.items[article.ID] = &copy
	return nil
}

func (r *fakeArticleRepo) Update(_ context.Context, _ uuid.UUID, article *domain.Article) error {
	copy := *article
	r.items[article.ID] = &copy
	return nil
}

func (r *fakeArticleRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}

func (r *fakeArticleRepo) ListSince(_ context.Context, _ uuid.UUID, _ int64) ([]domain.Article, error) {
	return nil, nil
}

func (r *fakeArticleRepo) ApplyRemote(_ context.Context, companyID uuid.UUID, article *domain.Article) error {
	return r.Create(context.Background(), companyID, article)
}
