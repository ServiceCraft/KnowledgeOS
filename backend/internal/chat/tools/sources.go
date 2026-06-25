package tools

import (
	"strconv"

	"github.com/knowledgeos/backend/internal/domain"
)

// sourcesFromCandidates converts retriever results into chat sources, capping
// the number kept so tool output stays bounded.
func sourcesFromCandidates(items []domain.RAGCandidate, limit int) []domain.ChatSource {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]domain.ChatSource, 0, len(items))
	for _, item := range items {
		out = append(out, domain.ChatSource{
			SourceID:   item.SourceID,
			EntityType: item.EntityType,
			EntityID:   item.EntityID,
			ChunkIdx:   item.ChunkIdx,
			Title:      item.Title,
			Content:    item.Content,
			Snippet:    item.Snippet,
			Score:      item.Score,
		})
	}
	return out
}

func entitySourceID(entityType domain.KBEntityType, id string) string {
	return string(entityType) + ":" + id + ":" + strconv.Itoa(0)
}
