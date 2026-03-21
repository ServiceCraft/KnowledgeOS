package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type PricingService struct {
	pricing domain.PricingNodeRepository
}

func NewPricingService(pricing domain.PricingNodeRepository) *PricingService {
	return &PricingService{pricing: pricing}
}

func (s *PricingService) List(ctx context.Context, companyID uuid.UUID, filter domain.PricingNodeFilter) ([]domain.PricingNode, int64, error) {
	return s.pricing.List(ctx, companyID, filter)
}

func (s *PricingService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.PricingNode, error) {
	return s.pricing.GetByID(ctx, companyID, id)
}

func (s *PricingService) Create(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	if node.Name == "" {
		return errors.New("name is required")
	}
	if node.ParentID != nil {
		if _, err := s.pricing.GetByID(ctx, companyID, *node.ParentID); err != nil {
			return errors.New("parent node not found")
		}
	}
	return s.pricing.Create(ctx, companyID, node)
}

func (s *PricingService) Update(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	if _, err := s.pricing.GetByID(ctx, companyID, node.ID); err != nil {
		return errors.New("pricing node not found")
	}
	return s.pricing.Update(ctx, companyID, node)
}

func (s *PricingService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.pricing.GetByID(ctx, companyID, id); err != nil {
		return errors.New("pricing node not found")
	}
	return s.pricing.Delete(ctx, companyID, id)
}

func (s *PricingService) Move(ctx context.Context, companyID uuid.UUID, id uuid.UUID, newParentID *uuid.UUID) error {
	node, err := s.pricing.GetByID(ctx, companyID, id)
	if err != nil {
		return errors.New("pricing node not found")
	}

	if newParentID != nil {
		if *newParentID == id {
			return errors.New("cannot move node under itself")
		}
		if _, err := s.pricing.GetByID(ctx, companyID, *newParentID); err != nil {
			return errors.New("new parent not found")
		}
		if err := s.checkCycle(ctx, companyID, id, *newParentID); err != nil {
			return err
		}
	}

	node.ParentID = newParentID
	return s.pricing.Update(ctx, companyID, node)
}

func (s *PricingService) checkCycle(ctx context.Context, companyID uuid.UUID, nodeID, targetParentID uuid.UUID) error {
	visited := map[uuid.UUID]bool{nodeID: true}
	current := targetParentID

	for {
		if visited[current] {
			return errors.New("move would create a cycle")
		}
		visited[current] = true

		parent, err := s.pricing.GetByID(ctx, companyID, current)
		if err != nil {
			break
		}
		if parent.ParentID == nil {
			break
		}
		current = *parent.ParentID
	}
	return nil
}
