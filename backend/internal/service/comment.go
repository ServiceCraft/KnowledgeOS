package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type CommentService struct {
	comments domain.CommentRepository
	qa       domain.QAPairRepository
	articles domain.ArticleRepository
	pricing  domain.PricingNodeRepository
}

func NewCommentService(comments domain.CommentRepository, qa domain.QAPairRepository, articles domain.ArticleRepository, pricing domain.PricingNodeRepository) *CommentService {
	return &CommentService{comments: comments, qa: qa, articles: articles, pricing: pricing}
}

func (s *CommentService) List(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID, filter domain.CommentFilter) ([]domain.Comment, int64, error) {
	return s.comments.List(ctx, companyID, entityType, entityID, filter)
}

func (s *CommentService) Create(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	if comment.Body == "" {
		return errors.New("body is required")
	}
	if err := s.validateEntity(ctx, companyID, comment.EntityType, comment.EntityID); err != nil {
		return err
	}
	return s.comments.Create(ctx, companyID, comment)
}

func (s *CommentService) Update(ctx context.Context, companyID uuid.UUID, comment *domain.Comment) error {
	if _, err := s.comments.GetByID(ctx, companyID, comment.ID); err != nil {
		return errors.New("comment not found")
	}
	return s.comments.Update(ctx, companyID, comment)
}

func (s *CommentService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.comments.GetByID(ctx, companyID, id); err != nil {
		return errors.New("comment not found")
	}
	return s.comments.Delete(ctx, companyID, id)
}

func (s *CommentService) validateEntity(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID) error {
	switch entityType {
	case "qa":
		_, err := s.qa.GetByID(ctx, companyID, entityID)
		if err != nil {
			return errors.New("target qa pair not found")
		}
	case "article":
		_, err := s.articles.GetByID(ctx, companyID, entityID)
		if err != nil {
			return errors.New("target article not found")
		}
	case "pricing":
		_, err := s.pricing.GetByID(ctx, companyID, entityID)
		if err != nil {
			return errors.New("target pricing node not found")
		}
	default:
		return errors.New("invalid entity type")
	}
	return nil
}
