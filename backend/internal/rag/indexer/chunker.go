package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

const (
	DefaultEmbeddingDim = 256
	targetArticleChunk  = 1200
	minArticleChunk     = 350
)

var whitespace = regexp.MustCompile(`\s+`)

type Chunker struct{}

// NewChunker executes the indexer.NewChunker operation.
func NewChunker() *Chunker {
	return &Chunker{}
}

// ChunksForArticle executes the indexer.Chunker.ChunksForArticle operation.
func (c *Chunker) ChunksForArticle(companyID uuid.UUID, article *domain.Article) []domain.RAGChunk {
	title := strings.TrimSpace(article.Title)
	paragraphs := splitParagraphs(article.Body)
	if len(paragraphs) == 0 && title == "" {
		return nil
	}

	var chunks []domain.RAGChunk
	var buf []string
	currentLen := 0
	flush := func() {
		if len(buf) == 0 {
			return
		}
		content := strings.TrimSpace(strings.Join(buf, "\n\n"))
		if title != "" {
			content = title + "\n\n" + content
		}
		chunks = append(chunks, c.chunk(companyID, domain.KBEntityArticle, article.ID, len(chunks), nil, title, content, content))
		buf = nil
		currentLen = 0
	}

	for _, p := range paragraphs {
		if currentLen > 0 && currentLen+len(p) > targetArticleChunk && currentLen >= minArticleChunk {
			flush()
		}
		buf = append(buf, p)
		currentLen += len(p)
	}
	flush()

	if len(chunks) == 0 && title != "" {
		chunks = append(chunks, c.chunk(companyID, domain.KBEntityArticle, article.ID, 0, nil, title, title, title))
	}
	return chunks
}

// ChunksForQA executes the indexer.Chunker.ChunksForQA operation.
func (c *Chunker) ChunksForQA(companyID uuid.UUID, qa *domain.QAPair) []domain.RAGChunk {
	question := strings.TrimSpace(qa.Question)
	answer := strings.TrimSpace(qa.Answer)
	if question == "" && answer == "" {
		return nil
	}
	content := strings.TrimSpace("Вопрос: " + question + "\nОтвет: " + answer)
	embeddingText := question
	if embeddingText == "" {
		embeddingText = content
	}
	return []domain.RAGChunk{
		c.chunk(companyID, domain.KBEntityQA, qa.ID, 0, qa.ThemeID, question, content, embeddingText),
	}
}

// ChunksForPricing executes the indexer.Chunker.ChunksForPricing operation.
func (c *Chunker) ChunksForPricing(companyID uuid.UUID, node *domain.PricingNode, parentPath []string) []domain.RAGChunk {
	name := strings.TrimSpace(node.Name)
	if name == "" {
		return nil
	}
	path := append([]string{}, parentPath...)
	path = append(path, name)
	content := "Услуга: " + strings.Join(path, " / ")
	if node.NodeType != "" {
		content += fmt.Sprintf("\nТип: %s", node.NodeType)
	}
	if node.Price != nil {
		content += fmt.Sprintf("\nЦена: %.2f", *node.Price)
	}
	return []domain.RAGChunk{
		c.chunk(companyID, domain.KBEntityPricing, node.ID, 0, nil, name, content, content),
	}
}

func (c *Chunker) chunk(companyID uuid.UUID, entityType domain.KBEntityType, entityID uuid.UUID, idx int, themeID *uuid.UUID, title, content, embeddingText string) domain.RAGChunk {
	normalized := normalize(content)
	return domain.RAGChunk{
		CompanyID:     companyID,
		EntityType:    entityType,
		EntityID:      entityID,
		ChunkIdx:      idx,
		ThemeID:       themeID,
		Title:         strings.TrimSpace(title),
		Content:       strings.TrimSpace(content),
		EmbeddingText: strings.TrimSpace(embeddingText),
		ContentHash:   hashText(normalized),
		Dim:           DefaultEmbeddingDim,
	}
}

func splitParagraphs(text string) []string {
	lines := strings.Split(text, "\n")
	var out []string
	var buf []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(buf) > 0 {
				out = append(out, strings.Join(buf, "\n"))
				buf = nil
			}
			continue
		}
		buf = append(buf, line)
	}
	if len(buf) > 0 {
		out = append(out, strings.Join(buf, "\n"))
	}
	return out
}

func normalize(text string) string {
	return strings.TrimSpace(whitespace.ReplaceAllString(text, " "))
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
