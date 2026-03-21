package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type LinkService struct {
	links    domain.EntityLinkRepository
	qa       domain.QAPairRepository
	articles domain.ArticleRepository
	pricing  domain.PricingNodeRepository
}

func NewLinkService(links domain.EntityLinkRepository, qa domain.QAPairRepository, articles domain.ArticleRepository, pricing domain.PricingNodeRepository) *LinkService {
	return &LinkService{links: links, qa: qa, articles: articles, pricing: pricing}
}

func (s *LinkService) List(ctx context.Context, companyID uuid.UUID, sourceType string, sourceID uuid.UUID, filter domain.EntityLinkFilter) ([]domain.EntityLink, int64, error) {
	return s.links.List(ctx, companyID, sourceType, sourceID, filter)
}

func (s *LinkService) Create(ctx context.Context, companyID uuid.UUID, link *domain.EntityLink) error {
	if err := s.validateEntity(ctx, companyID, link.SourceType, link.SourceID); err != nil {
		return errors.New("source entity not found")
	}
	if link.TargetType != nil && link.TargetID != nil {
		if err := s.validateEntity(ctx, companyID, *link.TargetType, *link.TargetID); err != nil {
			return errors.New("target entity not found")
		}
	}
	return s.links.Create(ctx, companyID, link)
}

func (s *LinkService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.links.GetByID(ctx, companyID, id); err != nil {
		return errors.New("link not found")
	}
	return s.links.Delete(ctx, companyID, id)
}

func (s *LinkService) validateEntity(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID) error {
	switch entityType {
	case "qa":
		_, err := s.qa.GetByID(ctx, companyID, entityID)
		return err
	case "article":
		_, err := s.articles.GetByID(ctx, companyID, entityID)
		return err
	case "pricing":
		_, err := s.pricing.GetByID(ctx, companyID, entityID)
		return err
	}
	return errors.New("invalid entity type")
}
