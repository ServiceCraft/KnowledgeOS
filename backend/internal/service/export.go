package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type ExportService struct {
	db       *gorm.DB
	themes   domain.ThemeRepository
	qa       domain.QAPairRepository
	pricing  domain.PricingNodeRepository
	articles domain.ArticleRepository
	comments domain.CommentRepository
	links    domain.EntityLinkRepository
}

func NewExportService(db *gorm.DB, themes domain.ThemeRepository, qa domain.QAPairRepository, pricing domain.PricingNodeRepository, articles domain.ArticleRepository, comments domain.CommentRepository, links domain.EntityLinkRepository) *ExportService {
	return &ExportService{db: db, themes: themes, qa: qa, pricing: pricing, articles: articles, comments: comments, links: links}
}

type ExportData struct {
	Themes       []domain.Theme       `json:"themes"`
	QAPairs      []domain.QAPair      `json:"qa_pairs"`
	PricingNodes []domain.PricingNode `json:"pricing_nodes"`
	Articles     []domain.Article     `json:"articles"`
}

func (s *ExportService) Export(ctx context.Context, companyID uuid.UUID) (*ExportData, error) {
	bigLimit := domain.ThemeFilter{Page: 1, Limit: 200}
	themes, _, err := s.themes.List(ctx, companyID, bigLimit)
	if err != nil {
		return nil, err
	}

	qas, _, err := s.qa.List(ctx, companyID, domain.QAPairFilter{Page: 1, Limit: 200})
	if err != nil {
		return nil, err
	}

	nodes, _, err := s.pricing.List(ctx, companyID, domain.PricingNodeFilter{Page: 1, Limit: 200})
	if err != nil {
		return nil, err
	}

	articles, _, err := s.articles.List(ctx, companyID, domain.ArticleFilter{Page: 1, Limit: 200})
	if err != nil {
		return nil, err
	}

	return &ExportData{
		Themes:       themes,
		QAPairs:      qas,
		PricingNodes: nodes,
		Articles:     articles,
	}, nil
}

type ImportData struct {
	Themes       []domain.Theme       `json:"themes"`
	QAPairs      []domain.QAPair      `json:"qa_pairs"`
	PricingNodes []domain.PricingNode `json:"pricing_nodes"`
	Articles     []domain.Article     `json:"articles"`
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (s *ExportService) Import(ctx context.Context, companyID uuid.UUID, data *ImportData) (*ImportResult, error) {
	if data == nil {
		return nil, errors.New("no data to import")
	}

	result := &ImportResult{Errors: []string{}}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		omitCols := []string{"search_vector", "SearchVector"}

		for i := range data.Themes {
			ensureID(&data.Themes[i].BaseModel)
			data.Themes[i].CompanyID = companyID
			if err := tx.Omit(omitCols...).Where("id = ? AND company_id = ?", data.Themes[i].ID, companyID).
				Assign(&data.Themes[i]).FirstOrCreate(&data.Themes[i]).Error; err != nil {
				return fmt.Errorf("theme %q: %w", data.Themes[i].Name, err)
			}
			result.Imported++
		}

		for i := range data.QAPairs {
			ensureID(&data.QAPairs[i].BaseModel)
			data.QAPairs[i].CompanyID = companyID
			if err := tx.Omit(omitCols...).Where("id = ? AND company_id = ?", data.QAPairs[i].ID, companyID).
				Assign(&data.QAPairs[i]).FirstOrCreate(&data.QAPairs[i]).Error; err != nil {
				return fmt.Errorf("qa %q: %w", data.QAPairs[i].Question, err)
			}
			result.Imported++
		}

		// Insert pricing nodes without parent_id first, then update parent_id
		parentIDs := make(map[uuid.UUID]*uuid.UUID)
		for i := range data.PricingNodes {
			ensureID(&data.PricingNodes[i].BaseModel)
			data.PricingNodes[i].CompanyID = companyID
			parentIDs[data.PricingNodes[i].ID] = data.PricingNodes[i].ParentID
			data.PricingNodes[i].ParentID = nil
			if err := tx.Omit(omitCols...).Where("id = ? AND company_id = ?", data.PricingNodes[i].ID, companyID).
				Assign(&data.PricingNodes[i]).FirstOrCreate(&data.PricingNodes[i]).Error; err != nil {
				return fmt.Errorf("pricing %q: %w", data.PricingNodes[i].Name, err)
			}
			result.Imported++
		}
		// Restore parent_id references
		for id, pid := range parentIDs {
			if pid != nil {
				if err := tx.Model(&domain.PricingNode{}).Where("id = ? AND company_id = ?", id, companyID).
					Update("parent_id", pid).Error; err != nil {
					return fmt.Errorf("pricing parent update: %w", err)
				}
			}
		}

		for i := range data.Articles {
			ensureID(&data.Articles[i].BaseModel)
			data.Articles[i].CompanyID = companyID
			if err := tx.Omit(omitCols...).Where("id = ? AND company_id = ?", data.Articles[i].ID, companyID).
				Assign(&data.Articles[i]).FirstOrCreate(&data.Articles[i]).Error; err != nil {
				return fmt.Errorf("article %q: %w", data.Articles[i].Title, err)
			}
			result.Imported++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func ensureID(m *domain.BaseModel) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
}

func topSortPricing(nodes []domain.PricingNode) []domain.PricingNode {
	sorted := make([]domain.PricingNode, 0, len(nodes))
	inserted := make(map[uuid.UUID]bool)
	remaining := nodes
	for len(remaining) > 0 {
		var next []domain.PricingNode
		for _, n := range remaining {
			if n.ParentID == nil || inserted[*n.ParentID] {
				sorted = append(sorted, n)
				inserted[n.ID] = true
			} else {
				next = append(next, n)
			}
		}
		if len(next) == len(remaining) {
			sorted = append(sorted, next...)
			break
		}
		remaining = next
	}
	return sorted
}
