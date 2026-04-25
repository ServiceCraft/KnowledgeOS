package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type CallService struct {
	calls    domain.CallRepository
	mentions domain.QACallMentionRepository
	qa       domain.QAPairRepository
}

func NewCallService(calls domain.CallRepository, mentions domain.QACallMentionRepository, qa domain.QAPairRepository) *CallService {
	return &CallService{calls: calls, mentions: mentions, qa: qa}
}

func (s *CallService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Call, error) {
	return s.calls.GetByID(ctx, companyID, id)
}

func (s *CallService) ListMentionsForQA(ctx context.Context, companyID uuid.UUID, qaID uuid.UUID, filter domain.QACallMentionFilter) ([]domain.QAPairCallMentionView, int64, error) {
	if _, err := s.qa.GetByID(ctx, companyID, qaID); err != nil {
		return nil, 0, errors.New("qa pair not found")
	}
	return s.mentions.ListByQA(ctx, companyID, qaID, filter)
}
