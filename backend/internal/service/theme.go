package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type ThemeService struct {
	themes domain.ThemeRepository
	qa     domain.QAPairRepository
}

func NewThemeService(themes domain.ThemeRepository, qa domain.QAPairRepository) *ThemeService {
	return &ThemeService{themes: themes, qa: qa}
}

func (s *ThemeService) List(ctx context.Context, companyID uuid.UUID, filter domain.ThemeFilter) ([]domain.Theme, int64, error) {
	return s.themes.List(ctx, companyID, filter)
}

func (s *ThemeService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Theme, error) {
	return s.themes.GetByID(ctx, companyID, id)
}

func (s *ThemeService) Create(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	if theme.Name == "" {
		return errors.New("name is required")
	}
	return s.themes.Create(ctx, companyID, theme)
}

func (s *ThemeService) Update(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	if _, err := s.themes.GetByID(ctx, companyID, theme.ID); err != nil {
		return errors.New("theme not found")
	}
	return s.themes.Update(ctx, companyID, theme)
}

func (s *ThemeService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	if _, err := s.themes.GetByID(ctx, companyID, id); err != nil {
		return errors.New("theme not found")
	}
	return s.themes.Delete(ctx, companyID, id)
}

func (s *ThemeService) ListQA(ctx context.Context, companyID uuid.UUID, themeID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	if _, err := s.themes.GetByID(ctx, companyID, themeID); err != nil {
		return nil, 0, errors.New("theme not found")
	}
	filter.ThemeID = &themeID
	return s.qa.List(ctx, companyID, filter)
}
