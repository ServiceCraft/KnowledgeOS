package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type ArticleService struct {
	articles domain.ArticleRepository
}

func NewArticleService(articles domain.ArticleRepository) *ArticleService {
	return &ArticleService{articles: articles}
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
	return s.articles.Create(ctx, companyID, article)
}

func (s *ArticleService) Update(ctx context.Context, companyID uuid.UUID, article *domain.Article) error {
	if _, err := s.articles.GetByID(ctx, companyID, article.ID); err != nil {
		return errors.New("article not found")
	}
	return s.articles.Update(ctx, companyID, article)
}

func (s *ArticleService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.articles.GetByID(ctx, companyID, id); err != nil {
		return errors.New("article not found")
	}
	return s.articles.Delete(ctx, companyID, id)
}
