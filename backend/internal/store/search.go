package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SearchStore struct {
	*Store
}

func NewSearchStore(s *Store) *SearchStore {
	return &SearchStore{Store: s}
}

func (s *SearchStore) Search(ctx context.Context, companyID uuid.UUID, filter domain.SearchFilter) ([]domain.SearchResult, int64, error) {
	if filter.Query == "" {
		return nil, 0, nil
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	typeSet := map[string]bool{}
	for _, t := range filter.Types {
		typeSet[t] = true
	}

	var unions []string
	var args []interface{}

	// Word-similarity search: matches short query against substrings in long fields
	// word_similarity() finds the best matching window, ideal for type-ahead search
	if len(typeSet) == 0 || typeSet["qa"] {
		sub := `SELECT 'qa' AS entity_type, id AS entity_id, question AS title, LEFT(answer, 200) AS snippet,
				GREATEST(
					word_similarity(?, question),
					word_similarity(?, answer) * 0.8
				) AS rank
				FROM qa_pairs WHERE company_id = ? AND deleted_at IS NULL
				AND (? <% question OR ? <% answer)`
		queryArgs := []interface{}{filter.Query, filter.Query, companyID, filter.Query, filter.Query}
		if filter.ThemeID != nil {
			sub += " AND theme_id = ?"
			queryArgs = append(queryArgs, *filter.ThemeID)
		}
		unions = append(unions, sub)
		args = append(args, queryArgs...)
	}

	if len(typeSet) == 0 || typeSet["article"] {
		sub := `SELECT 'article' AS entity_type, id AS entity_id, title AS title, LEFT(body, 200) AS snippet,
				GREATEST(
					word_similarity(?, title),
					word_similarity(?, body) * 0.8
				) AS rank
				FROM articles WHERE company_id = ? AND deleted_at IS NULL
				AND (? <% title OR ? <% body)`
		args = append(args, filter.Query, filter.Query, companyID, filter.Query, filter.Query)
		unions = append(unions, sub)
	}

	if len(unions) == 0 {
		return nil, 0, nil
	}

	combined := strings.Join(unions, " UNION ALL ")

	// Use a transaction so SET LOCAL applies to the same connection as the queries
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer tx.Rollback()

	tx.Exec("SET LOCAL pg_trgm.word_similarity_threshold = 0.15")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS combined", combined)
	var total int64
	if err := tx.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	dataSQL := fmt.Sprintf("SELECT * FROM (%s) AS combined ORDER BY rank DESC LIMIT ? OFFSET ?", combined)
	dataArgs := append(args, filter.Limit, offset)

	var results []domain.SearchResult
	if err := tx.Raw(dataSQL, dataArgs...).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	tx.Commit()
	return results, total, nil
}
