package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type QAService struct {
	qa     domain.QAPairRepository
	themes domain.ThemeRepository
}

func NewQAService(qa domain.QAPairRepository, themes domain.ThemeRepository) *QAService {
	return &QAService{qa: qa, themes: themes}
}

func (s *QAService) List(ctx context.Context, companyID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	return s.qa.List(ctx, companyID, filter)
}

func (s *QAService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.QAPair, error) {
	return s.qa.GetByID(ctx, companyID, id)
}

func (s *QAService) Create(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	if qa.Question == "" || qa.Answer == "" {
		return errors.New("question and answer are required")
	}
	if qa.ThemeID != nil {
		if _, err := s.themes.GetByID(ctx, companyID, *qa.ThemeID); err != nil {
			return errors.New("theme not found")
		}
	}
	return s.qa.Create(ctx, companyID, qa)
}

func (s *QAService) Update(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	existing, err := s.qa.GetByID(ctx, companyID, qa.ID)
	if err != nil {
		return errors.New("qa pair not found")
	}
	if qa.ThemeID != nil {
		if _, err := s.themes.GetByID(ctx, companyID, *qa.ThemeID); err != nil {
			return errors.New("theme not found")
		}
	}
	_ = existing
	return s.qa.Update(ctx, companyID, qa)
}

func (s *QAService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.qa.GetByID(ctx, companyID, id); err != nil {
		return errors.New("qa pair not found")
	}
	return s.qa.Delete(ctx, companyID, id)
}
