package service

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SearchService struct {
	search domain.SearchRepository
}

// NewSearchService executes the service.NewSearchService operation.
func NewSearchService(search domain.SearchRepository) *SearchService {
	return &SearchService{search: search}
}

// Search executes the service.SearchService.Search operation.
func (s *SearchService) Search(ctx context.Context, companyID uuid.UUID, filter domain.SearchFilter) ([]domain.SearchResult, int64, error) {
	applog.TraceCall(ctx, "service.SearchService.Search")
	return s.search.Search(ctx, companyID, filter)
}
