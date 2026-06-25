package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type PricingService struct {
	pricing domain.PricingNodeRepository
	indexer domain.KnowledgeIndexScheduler
}

// NewPricingService executes the service.NewPricingService operation.
func NewPricingService(pricing domain.PricingNodeRepository, schedulers ...domain.KnowledgeIndexScheduler) *PricingService {
	var indexer domain.KnowledgeIndexScheduler
	if len(schedulers) > 0 {
		indexer = schedulers[0]
	}
	return &PricingService{pricing: pricing, indexer: indexer}
}

// List executes the service.PricingService.List operation.
func (s *PricingService) List(ctx context.Context, companyID uuid.UUID, filter domain.PricingNodeFilter) ([]domain.PricingNode, int64, error) {
	applog.TraceCall(ctx, "service.PricingService.List")
	return s.pricing.List(ctx, companyID, filter)
}

// GetByID executes the service.PricingService.GetByID operation.
func (s *PricingService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.PricingNode, error) {
	applog.TraceCall(ctx, "service.PricingService.GetByID")
	return s.pricing.GetByID(ctx, companyID, id)
}

// Create executes the service.PricingService.Create operation.
func (s *PricingService) Create(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	applog.TraceCall(ctx, "service.PricingService.Create")
	if node.Name == "" {
		return errors.New("name is required")
	}
	if node.ParentID != nil {
		if _, err := s.pricing.GetByID(ctx, companyID, *node.ParentID); err != nil {
			return errors.New("parent node not found")
		}
	}
	if err := s.pricing.Create(ctx, companyID, node); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, node.ID)
	return nil
}

// Update executes the service.PricingService.Update operation.
func (s *PricingService) Update(ctx context.Context, companyID uuid.UUID, node *domain.PricingNode) error {
	applog.TraceCall(ctx, "service.PricingService.Update")
	if _, err := s.pricing.GetByID(ctx, companyID, node.ID); err != nil {
		return errors.New("pricing node not found")
	}
	if err := s.pricing.Update(ctx, companyID, node); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, node.ID)
	return nil
}

// Delete executes the service.PricingService.Delete operation.
func (s *PricingService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.PricingService.Delete")
	if _, err := s.pricing.GetByID(ctx, companyID, id); err != nil {
		return errors.New("pricing node not found")
	}
	if err := s.pricing.Delete(ctx, companyID, id); err != nil {
		return err
	}
	s.scheduleDelete(ctx, companyID, id)
	return nil
}

// Move executes the service.PricingService.Move operation.
func (s *PricingService) Move(ctx context.Context, companyID uuid.UUID, id uuid.UUID, newParentID *uuid.UUID) error {
	applog.TraceCall(ctx, "service.PricingService.Move")
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
	if err := s.pricing.Update(ctx, companyID, node); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, node.ID)
	return nil
}

func (s *PricingService) checkCycle(ctx context.Context, companyID uuid.UUID, nodeID, targetParentID uuid.UUID) error {
	applog.TraceCall(ctx, "service.PricingService.checkCycle")
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

func (s *PricingService) scheduleUpsert(ctx context.Context, companyID, id uuid.UUID) {
	applog.TraceCall(ctx, "service.PricingService.scheduleUpsert")
	if s.indexer != nil {
		_ = s.indexer.ScheduleUpsert(ctx, companyID, domain.KBEntityPricing, id)
	}
}

func (s *PricingService) scheduleDelete(ctx context.Context, companyID, id uuid.UUID) {
	applog.TraceCall(ctx, "service.PricingService.scheduleDelete")
	if s.indexer != nil {
		_ = s.indexer.ScheduleDelete(ctx, companyID, domain.KBEntityPricing, id)
	}
}
