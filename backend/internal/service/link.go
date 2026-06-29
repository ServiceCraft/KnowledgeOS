package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type LinkService struct {
	links    domain.EntityLinkRepository
	qa       domain.QAPairRepository
	articles domain.ArticleRepository
	pricing  domain.PricingNodeRepository
}

// NewLinkService executes the service.NewLinkService operation.
func NewLinkService(links domain.EntityLinkRepository, qa domain.QAPairRepository, articles domain.ArticleRepository, pricing domain.PricingNodeRepository) *LinkService {
	return &LinkService{links: links, qa: qa, articles: articles, pricing: pricing}
}

// List executes the service.LinkService.List operation.
func (s *LinkService) List(ctx context.Context, companyID uuid.UUID, sourceType string, sourceID uuid.UUID, filter domain.EntityLinkFilter) ([]domain.EntityLink, int64, error) {
	applog.TraceCall(ctx, "service.LinkService.List")
	return s.links.List(ctx, companyID, sourceType, sourceID, filter)
}

// Create executes the service.LinkService.Create operation.
func (s *LinkService) Create(ctx context.Context, companyID uuid.UUID, link *domain.EntityLink) error {
	applog.TraceCall(ctx, "service.LinkService.Create")
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

// Delete executes the service.LinkService.Delete operation.
func (s *LinkService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.LinkService.Delete")
	if _, err := s.links.GetByID(ctx, companyID, id); err != nil {
		return errors.New("link not found")
	}
	return s.links.Delete(ctx, companyID, id)
}

func (s *LinkService) validateEntity(ctx context.Context, companyID uuid.UUID, entityType string, entityID uuid.UUID) error {
	applog.TraceCall(ctx, "service.LinkService.validateEntity")
	switch entityType {
	case "qa":
		_, err := s.qa.GetByID(ctx, companyID, entityID)
		return applog.TraceErr(ctx, "link: resolve qa entity", err)
	case "article":
		_, err := s.articles.GetByID(ctx, companyID, entityID)
		return applog.TraceErr(ctx, "link: resolve article entity", err)
	case "pricing":
		_, err := s.pricing.GetByID(ctx, companyID, entityID)
		return applog.TraceErr(ctx, "link: resolve pricing entity", err)
	}
	return errors.New("invalid entity type")
}
