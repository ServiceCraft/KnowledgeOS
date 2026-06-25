package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type CallService struct {
	calls    domain.CallRepository
	mentions domain.QACallMentionRepository
	qa       domain.QAPairRepository
}

// NewCallService executes the service.NewCallService operation.
func NewCallService(calls domain.CallRepository, mentions domain.QACallMentionRepository, qa domain.QAPairRepository) *CallService {
	return &CallService{calls: calls, mentions: mentions, qa: qa}
}

// GetByID executes the service.CallService.GetByID operation.
func (s *CallService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Call, error) {
	applog.TraceCall(ctx, "service.CallService.GetByID")
	return s.calls.GetByID(ctx, companyID, id)
}

// ListMentionsForQA executes the service.CallService.ListMentionsForQA operation.
func (s *CallService) ListMentionsForQA(ctx context.Context, companyID uuid.UUID, qaID uuid.UUID, filter domain.QACallMentionFilter) ([]domain.QAPairCallMentionView, int64, error) {
	applog.TraceCall(ctx, "service.CallService.ListMentionsForQA")
	if _, err := s.qa.GetByID(ctx, companyID, qaID); err != nil {
		return nil, 0, errors.New("qa pair not found")
	}
	return s.mentions.ListByQA(ctx, companyID, qaID, filter)
}
