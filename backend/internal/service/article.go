package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type ArticleService struct {
	articles domain.ArticleRepository
	indexer  domain.KnowledgeIndexScheduler
}

func NewArticleService(articles domain.ArticleRepository, schedulers ...domain.KnowledgeIndexScheduler) *ArticleService {
	var indexer domain.KnowledgeIndexScheduler
	if len(schedulers) > 0 {
		indexer = schedulers[0]
	}
	return &ArticleService{articles: articles, indexer: indexer}
}

func (s *ArticleService) List(ctx context.Context, companyID uuid.UUID, filter domain.ArticleFilter) ([]domain.Article, int64, error) {
	return s.articles.List(ctx, companyID, filter)
}

func (s *ArticleService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Article, error) {
	return s.articles.GetByID(ctx, companyID, id)
}

func (s *ArticleService) Create(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	if article.Title == "" {
		return errors.New("title is required")
	}
	if err := s.articles.Create(ctx, companyID, article); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, article.ID)
	return nil
}

func (s *ArticleService) Update(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	if _, err := s.articles.GetByID(ctx, companyID, article.ID); err != nil {
		return errors.New("article not found")
	}
	if err := s.articles.Update(ctx, companyID, article); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, article.ID)
	return nil
}

func (s *ArticleService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.articles.GetByID(ctx, companyID, id); err != nil {
		return errors.New("article not found")
	}
	if err := s.articles.Delete(ctx, companyID, id); err != nil {
		return err
	}
	s.scheduleDelete(ctx, companyID, id)
	return nil
}

func (s *ArticleService) scheduleUpsert(ctx context.Context, companyID, id uuid.UUID) {
	if s.indexer != nil {
		_ = s.indexer.ScheduleUpsert(ctx, companyID, domain.KBEntityArticle, id)
	}
}

func (s *ArticleService) scheduleDelete(ctx context.Context, companyID, id uuid.UUID) {
	if s.indexer != nil {
		_ = s.indexer.ScheduleDelete(ctx, companyID, domain.KBEntityArticle, id)
	}
}
