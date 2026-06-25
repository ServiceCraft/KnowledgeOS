package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type QAService struct {
	qa      domain.QAPairRepository
	themes  domain.ThemeRepository
	indexer domain.KnowledgeIndexScheduler
}

func NewQAService(qa domain.QAPairRepository, themes domain.ThemeRepository, schedulers ...domain.KnowledgeIndexScheduler) *QAService {
	var indexer domain.KnowledgeIndexScheduler
	if len(schedulers) > 0 {
		indexer = schedulers[0]
	}
	return &QAService{qa: qa, themes: themes, indexer: indexer}
}

func (s *QAService) List(ctx context.Context, companyID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	return s.qa.List(ctx, companyID, filter)
}

func (s *QAService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.QAPair, error) {
	return s.qa.GetByID(ctx, companyID, id)
}

func (s *QAService) Create(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	if qa.Question == "" {
		return errors.New("question is required")
	}
	if qa.ThemeID != nil {
		if _, err := s.themes.GetByID(ctx, companyID, *qa.ThemeID); err != nil {
			return errors.New("theme not found")
		}
	}
	if err := s.qa.Create(ctx, companyID, qa); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, qa.ID)
	return nil
}

func (s *QAService) Update(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	if _, err := s.qa.GetByID(ctx, companyID, qa.ID); err != nil {
		return errors.New("qa pair not found")
	}
	if err := s.qa.Update(ctx, companyID, qa); err != nil {
		return err
	}
	s.scheduleUpsert(ctx, companyID, qa.ID)
	return nil
}

func (s *QAService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.qa.GetByID(ctx, companyID, id); err != nil {
		return errors.New("qa pair not found")
	}
	if err := s.qa.Delete(ctx, companyID, id); err != nil {
		return err
	}
	s.scheduleDelete(ctx, companyID, id)
	return nil
}

func (s *QAService) ReviewAIAnswer(ctx context.Context, companyID uuid.UUID, qaID uuid.UUID, userID uuid.UUID, action string, editedAnswer string) (*domain.QAPair, error) {
	qa, err := s.qa.GetByID(ctx, companyID, qaID)
	if err != nil {
		return nil, errors.New("qa pair not found")
	}

	if qa.AIStatus == nil || *qa.AIStatus != "pending" {
		return nil, errors.New("no pending AI answer to review")
	}

	now := time.Now()
	qa.AIReviewedBy = &userID
	qa.AIReviewedAt = &now
	qa.UpdatedBy = &userID

	switch action {
	case "accept":
		status := "accepted"
		qa.AIStatus = &status
		if qa.AIAnswer != nil {
			qa.Answer = *qa.AIAnswer
		}
	case "reject":
		status := "rejected"
		qa.AIStatus = &status
	case "edit":
		if editedAnswer == "" {
			return nil, errors.New("edited_answer is required for edit action")
		}
		status := "edited"
		qa.AIStatus = &status
		qa.Answer = editedAnswer
	default:
		return nil, errors.New("invalid action: must be accept, reject, or edit")
	}

	if err := s.qa.Update(ctx, companyID, qa); err != nil {
		return nil, err
	}
	s.scheduleUpsert(ctx, companyID, qa.ID)
	return qa, nil
}

func (s *QAService) scheduleUpsert(ctx context.Context, companyID, id uuid.UUID) {
	if s.indexer != nil {
		_ = s.indexer.ScheduleUpsert(ctx, companyID, domain.KBEntityQA, id)
	}
}

func (s *QAService) scheduleDelete(ctx context.Context, companyID, id uuid.UUID) {
	if s.indexer != nil {
		_ = s.indexer.ScheduleDelete(ctx, companyID, domain.KBEntityQA, id)
	}
}
