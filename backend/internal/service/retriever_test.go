package service

import (
	"context"
	"testing"

	"github.com/knowledgeos/backend/internal/llm"
)

func rewriteWith(t *testing.T, modelOutput string) string {
	t.Helper()
	provider := &fakeChatProvider{response: &llm.ChatResponse{
		Message: llm.ToolMessage{Role: llm.RoleAssistant, Content: modelOutput},
	}}
	svc := &RetrieverService{}
	return svc.rewriteQuery(context.Background(), provider, "надо ли готовить кота перед узи")
}

func TestRewriteQueryAcceptsNormalRewrite(t *testing.T) {
	if got := rewriteWith(t, "Подготовка кота к УЗИ"); got != "Подготовка кота к УЗИ" {
		t.Fatalf("rewriteQuery() = %q, want the rewritten query", got)
	}
}

func TestRewriteQueryDiscardsRefusals(t *testing.T) {
	refusals := []string{
		"Я не могу обсуждать эту тему. Давайте поговорим о чём-нибудь ещё.",
		"Извините, я не буду отвечать на этот вопрос.",
		"I can't help with that.",
		"",
	}
	for _, out := range refusals {
		if got := rewriteWith(t, out); got != "" {
			t.Fatalf("rewriteQuery(%q) = %q, want empty (fallback to original)", out, got)
		}
	}
}
