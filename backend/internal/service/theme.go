package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type ThemeService struct {
	themes domain.ThemeRepository
	qa     domain.QAPairRepository
}

// NewThemeService executes the service.NewThemeService operation.
func NewThemeService(themes domain.ThemeRepository, qa domain.QAPairRepository) *ThemeService {
	return &ThemeService{themes: themes, qa: qa}
}

// List executes the service.ThemeService.List operation.
func (s *ThemeService) List(ctx context.Context, companyID uuid.UUID, filter domain.ThemeFilter) ([]domain.Theme, int64, error) {
	applog.TraceCall(ctx, "service.ThemeService.List")
	return s.themes.List(ctx, companyID, filter)
}

// GetByID executes the service.ThemeService.GetByID operation.
func (s *ThemeService) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Theme, error) {
	applog.TraceCall(ctx, "service.ThemeService.GetByID")
	return s.themes.GetByID(ctx, companyID, id)
}

// Create executes the service.ThemeService.Create operation.
func (s *ThemeService) Create(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	applog.TraceCall(ctx, "service.ThemeService.Create")
	if theme.Name == "" {
		return errors.New("name is required")
	}
	return s.themes.Create(ctx, companyID, theme)
}

// Update executes the service.ThemeService.Update operation.
func (s *ThemeService) Update(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	applog.TraceCall(ctx, "service.ThemeService.Update")
	if _, err := s.themes.GetByID(ctx, companyID, theme.ID); err != nil {
		return errors.New("theme not found")
	}
	return s.themes.Update(ctx, companyID, theme)
}

// Delete executes the service.ThemeService.Delete operation.
func (s *ThemeService) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.ThemeService.Delete")
	if _, err := s.themes.GetByID(ctx, companyID, id); err != nil {
		return errors.New("theme not found")
	}
	return s.themes.Delete(ctx, companyID, id)
}

// ListQA executes the service.ThemeService.ListQA operation.
func (s *ThemeService) ListQA(ctx context.Context, companyID uuid.UUID, themeID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	applog.TraceCall(ctx, "service.ThemeService.ListQA")
	if _, err := s.themes.GetByID(ctx, companyID, themeID); err != nil {
		return nil, 0, errors.New("theme not found")
	}
	filter.ThemeID = &themeID
	return s.qa.List(ctx, companyID, filter)
}
