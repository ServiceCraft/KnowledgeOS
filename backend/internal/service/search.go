package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SearchService struct {
	search domain.SearchRepository
}

func NewSearchService(search domain.SearchRepository) *SearchService {
	return &SearchService{search: search}
}

func (s *SearchService) Search(ctx context.Context, companyID uuid.UUID, filter domain.SearchFilter) ([]domain.SearchResult, int64, error) {
	return s.search.Search(ctx, companyID, filter)
}
