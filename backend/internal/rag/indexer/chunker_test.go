package indexer

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

func TestChunkerChunksQA(t *testing.T) {
	companyID := uuid.New()
	themeID := uuid.New()
	qaID := uuid.New()
	chunker := NewChunker()

	chunks := chunker.ChunksForQA(companyID, &domain.QAPair{
		SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: qaID}},
		ThemeID:       &themeID,
		Question:      "Сколько стоит стерилизация кошки?",
		Answer:        "Стоимость стерилизации кошки — 3500 рублей.",
	})

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	chunk := chunks[0]
	if chunk.CompanyID != companyID || chunk.EntityType != domain.KBEntityQA || chunk.EntityID != qaID {
		t.Fatalf("unexpected chunk identity: %+v", chunk)
	}
	if chunk.ThemeID == nil || *chunk.ThemeID != themeID {
		t.Fatalf("ThemeID = %v, want %s", chunk.ThemeID, themeID)
	}
	if chunk.EmbeddingText != "Сколько стоит стерилизация кошки?" {
		t.Fatalf("EmbeddingText = %q", chunk.EmbeddingText)
	}
	if !strings.Contains(chunk.Content, "Стоимость стерилизации") {
		t.Fatalf("Content = %q", chunk.Content)
	}
	if chunk.ContentHash == "" || chunk.Dim != DefaultEmbeddingDim {
		t.Fatalf("missing hash or dim: %+v", chunk)
	}
}

func TestChunkerUsesReviewedAIAnswerWhenAnswerEmpty(t *testing.T) {
	chunker := NewChunker()
	aiAnswer := "Приём ведёт Гаврилова Анастасия Дмитриевна."

	for _, status := range []string{"accepted", "edited"} {
		t.Run(status, func(t *testing.T) {
			chunks := chunker.ChunksForQA(uuid.New(), &domain.QAPair{
				SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: uuid.New()}},
				Question:      "Есть ли гастроэнтеролог в клинике?",
				Answer:        "",
				AIAnswer:      &aiAnswer,
				AIStatus:      &status,
			})
			if len(chunks) != 1 {
				t.Fatalf("len(chunks) = %d, want 1", len(chunks))
			}
			if !strings.Contains(chunks[0].Content, "Гаврилова") {
				t.Fatalf("reviewed ai_answer missing from content: %q", chunks[0].Content)
			}
		})
	}
}

func TestChunkerIgnoresUnreviewedAIAnswer(t *testing.T) {
	chunker := NewChunker()
	aiAnswer := "Непроверенный черновик, который клиенту показывать нельзя."

	for _, status := range []string{"pending", "rejected"} {
		t.Run(status, func(t *testing.T) {
			chunks := chunker.ChunksForQA(uuid.New(), &domain.QAPair{
				SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: uuid.New()}},
				Question:      "Есть ли гастроэнтеролог в клинике?",
				Answer:        "",
				AIAnswer:      &aiAnswer,
				AIStatus:      &status,
			})
			// The question alone still gets indexed so the pair stays findable,
			// but the unreviewed draft must not reach the client.
			if len(chunks) == 1 && strings.Contains(chunks[0].Content, "Непроверенный") {
				t.Fatalf("unreviewed ai_answer leaked into content: %q", chunks[0].Content)
			}
		})
	}
}

func TestChunkerPrefersHumanAnswerOverAIAnswer(t *testing.T) {
	chunker := NewChunker()
	aiAnswer := "Черновик от ИИ."
	status := "accepted"

	chunks := chunker.ChunksForQA(uuid.New(), &domain.QAPair{
		SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: uuid.New()}},
		Question:      "Есть ли гастроэнтеролог в клинике?",
		Answer:        "Да, приём по предварительной записи.",
		AIAnswer:      &aiAnswer,
		AIStatus:      &status,
	})

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	if strings.Contains(chunks[0].Content, "Черновик") {
		t.Fatalf("ai_answer overrode the human answer: %q", chunks[0].Content)
	}
}

func TestChunkerSplitsArticleAndHashesStably(t *testing.T) {
	companyID := uuid.New()
	articleID := uuid.New()
	chunker := NewChunker()
	body := strings.Repeat("Первый абзац про вакцинацию.\n\n", 30)
	article := &domain.Article{
		SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: articleID}},
		Title:         "Вакцинация",
		Body:          body,
	}

	first := chunker.ChunksForArticle(companyID, article)
	second := chunker.ChunksForArticle(companyID, article)

	if len(first) < 2 {
		t.Fatalf("len(chunks) = %d, want split into multiple chunks", len(first))
	}
	if first[0].ContentHash != second[0].ContentHash {
		t.Fatalf("hash is not stable: %s != %s", first[0].ContentHash, second[0].ContentHash)
	}
	if !strings.Contains(first[0].Content, "Вакцинация") {
		t.Fatalf("title not included in content: %q", first[0].Content)
	}
}

func TestChunkerChunksPricing(t *testing.T) {
	companyID := uuid.New()
	nodeID := uuid.New()
	price := 3500.0
	chunker := NewChunker()

	chunks := chunker.ChunksForPricing(companyID, &domain.PricingNode{
		SyncableModel: domain.SyncableModel{BaseModel: domain.BaseModel{ID: nodeID}},
		NodeType:      "service",
		Name:          "Стерилизация кошки",
		Price:         &price,
	}, []string{"Хирургия"})

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	if !strings.Contains(chunks[0].Content, "Хирургия / Стерилизация кошки") {
		t.Fatalf("Content = %q", chunks[0].Content)
	}
	if !strings.Contains(chunks[0].Content, "3500.00") {
		t.Fatalf("price missing from content: %q", chunks[0].Content)
	}
}
