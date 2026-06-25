package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type KBEmbeddingStore struct {
	*Store
}

func NewKBEmbeddingStore(s *Store) *KBEmbeddingStore {
	return &KBEmbeddingStore{Store: s}
}

func (s *KBEmbeddingStore) UpsertChunks(ctx context.Context, companyID uuid.UUID, chunks []domain.RAGChunk) error {
	for _, chunk := range chunks {
		if chunk.CompanyID == uuid.Nil {
			chunk.CompanyID = companyID
		}
		if chunk.CompanyID != companyID {
			return fmt.Errorf("chunk company mismatch")
		}
		if len(chunk.Embedding) != chunk.Dim {
			return fmt.Errorf("embedding dim mismatch: got %d want %d", len(chunk.Embedding), chunk.Dim)
		}
		embedding := pgvectorLiteral(chunk.Embedding)
		if err := s.db.WithContext(ctx).Exec(`
			INSERT INTO kb_embeddings (
				company_id, entity_type, entity_id, chunk_idx, theme_id, title,
				content, content_hash, embedding, model, dim
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?::vector, ?, ?)
			ON CONFLICT (company_id, entity_type, entity_id, chunk_idx) DO UPDATE SET
				theme_id = EXCLUDED.theme_id,
				title = EXCLUDED.title,
				content = EXCLUDED.content,
				content_hash = EXCLUDED.content_hash,
				embedding = EXCLUDED.embedding,
				model = EXCLUDED.model,
				dim = EXCLUDED.dim,
				updated_at = now()
		`, companyID, chunk.EntityType, chunk.EntityID, chunk.ChunkIdx, chunk.ThemeID, chunk.Title,
			chunk.Content, chunk.ContentHash, embedding, chunk.Model, chunk.Dim).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *KBEmbeddingStore) DeleteEntity(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Where("company_id = ? AND entity_type = ? AND entity_id = ?", companyID, entityType, entityID).
		Delete(&domain.KBEmbedding{}).Error
}

func (s *KBEmbeddingStore) DeleteChunksExcept(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID, keepChunkIdx []int) error {
	q := s.db.WithContext(ctx).
		Where("company_id = ? AND entity_type = ? AND entity_id = ?", companyID, entityType, entityID)
	if len(keepChunkIdx) > 0 {
		q = q.Where("chunk_idx NOT IN ?", keepChunkIdx)
	}
	return q.Delete(&domain.KBEmbedding{}).Error
}

func (s *KBEmbeddingStore) DeleteCompany(ctx context.Context, companyID uuid.UUID) error {
	return s.db.WithContext(ctx).Where("company_id = ?", companyID).Delete(&domain.KBEmbedding{}).Error
}

func (s *KBEmbeddingStore) ListByEntity(ctx context.Context, companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID) ([]domain.KBEmbedding, error) {
	var items []domain.KBEmbedding
	if err := s.db.WithContext(ctx).
		Where("company_id = ? AND entity_type = ? AND entity_id = ?", companyID, entityType, entityID).
		Order("chunk_idx ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *KBEmbeddingStore) VectorSearch(ctx context.Context, companyID uuid.UUID, embedding []float32, filter domain.RetrieveRequest) ([]domain.RAGCandidate, error) {
	limit := filter.VectorTopK
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT
			(entity_type || ':' || entity_id::text || ':' || chunk_idx::text) AS source_id,
			entity_type, entity_id, chunk_idx, theme_id, title, content,
			LEFT(content, 240) AS snippet,
			1 - (embedding <=> ?::vector) AS score
		FROM kb_embeddings
		WHERE company_id = ?`
	args := []interface{}{pgvectorLiteral(embedding), companyID}
	if len(filter.Types) > 0 {
		query += " AND entity_type IN ?"
		args = append(args, kbTypesToStrings(filter.Types))
	}
	if filter.ThemeID != nil {
		query += " AND (theme_id = ? OR theme_id IS NULL)"
		args = append(args, *filter.ThemeID)
	}
	query += " ORDER BY embedding <=> ?::vector LIMIT ?"
	args = append(args, pgvectorLiteral(embedding), limit)

	var results []domain.RAGCandidate
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error; err != nil {
		return nil, err
	}
	for i := range results {
		results[i].VectorRank = i + 1
	}
	return results, nil
}

func (s *KBEmbeddingStore) CountByCompany(ctx context.Context, companyID uuid.UUID) (int64, error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&domain.KBEmbedding{}).
		Where("company_id = ?", companyID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func pgvectorLiteral(values []float32) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func kbTypesToStrings(types []domain.KBEntityType) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	return out
}
