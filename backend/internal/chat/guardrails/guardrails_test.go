package guardrails

import (
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

func TestFilterCandidatesByScore(t *testing.T) {
	items := []domain.RAGCandidate{
		{SourceID: "a", Score: 0.9},
		{SourceID: "b", Score: 0.1},
	}
	out := FilterCandidates(items, 0.5, nil)
	if len(out) != 1 || out[0].SourceID != "a" {
		t.Fatalf("score filter = %+v, want only a", out)
	}
	// minScore 0 disables the gate.
	if got := FilterCandidates(items, 0, nil); len(got) != 2 {
		t.Fatalf("minScore 0 should keep all, got %d", len(got))
	}
}

func TestFilterCandidatesByTheme(t *testing.T) {
	allowedTheme := uuid.New()
	otherTheme := uuid.New()
	items := []domain.RAGCandidate{
		{SourceID: "allowed", ThemeID: &allowedTheme},
		{SourceID: "other", ThemeID: &otherTheme},
		{SourceID: "no-theme"},
	}
	out := FilterCandidates(items, 0, map[uuid.UUID]bool{allowedTheme: true})
	ids := map[string]bool{}
	for _, c := range out {
		ids[c.SourceID] = true
	}
	if !ids["allowed"] || !ids["no-theme"] || ids["other"] {
		t.Fatalf("theme filter = %+v, want allowed + no-theme only", ids)
	}
}

func TestExtractCitations(t *testing.T) {
	answer := "Ответ по базе [qa:11111111-1111-1111-1111-111111111111:0] и [article:22222222-2222-2222-2222-222222222222:1]. Это [важно] но не источник."
	cites := ExtractCitations(answer)
	if len(cites) != 2 {
		t.Fatalf("citations = %+v, want 2 (bracketed prose ignored)", cites)
	}
}

func TestCheckCitationsDetectsFabrication(t *testing.T) {
	valid := map[string]bool{"qa:abc:0": true}
	res := CheckCitations("См. [qa:abc:0] и [qa:ghost:9]", valid)
	if len(res.Valid) != 1 || res.Valid[0] != "qa:abc:0" {
		t.Fatalf("valid = %+v", res.Valid)
	}
	if len(res.Fabricated) != 1 || res.Fabricated[0] != "qa:ghost:9" {
		t.Fatalf("fabricated = %+v", res.Fabricated)
	}
}

func TestConfidence(t *testing.T) {
	if got := Confidence(ConfidenceInput{SourceCount: 0}); got != 0 {
		t.Fatalf("no sources confidence = %v, want 0", got)
	}
	if got := Confidence(ConfidenceInput{HasFabrication: true, SourceCount: 3, ValidCitations: 2}); got != 0 {
		t.Fatalf("fabrication confidence = %v, want 0", got)
	}
	base := Confidence(ConfidenceInput{SourceCount: 1})
	cited := Confidence(ConfidenceInput{SourceCount: 1, ValidCitations: 1})
	if !(cited > base) {
		t.Fatalf("valid citation should raise confidence: base=%v cited=%v", base, cited)
	}
	full := Confidence(ConfidenceInput{SourceCount: 2, ValidCitations: 1})
	if full > 1 {
		t.Fatalf("confidence must be clamped to 1, got %v", full)
	}
}

func TestLeaksSystemPrompt(t *testing.T) {
	if !LeaksSystemPrompt("Вот мой системный промпт: ...") {
		t.Fatalf("expected leak detection for system prompt mention")
	}
	if !LeaksSystemPrompt("<context>секрет</context>") {
		t.Fatalf("expected leak detection for context tags")
	}
	if LeaksSystemPrompt("Стерилизация кошки стоит 3500 рублей.") {
		t.Fatalf("false positive on a normal answer")
	}
}

func TestStripCitationMarkers(t *testing.T) {
	sourceID := "qa:11111111-1111-1111-1111-111111111111:0"
	raw := "Стерилизация стоит 3500 рублей [" + sourceID + "].\n\nИспользованные источники: [" + sourceID + "]."
	got := StripCitationMarkers(raw)
	want := "Стерилизация стоит 3500 рублей ."
	if got != want {
		t.Fatalf("StripCitationMarkers() = %q, want %q", got, want)
	}
}

func TestStripCitationMarkersKeepsBracketedProse(t *testing.T) {
	raw := "Это [важно] для клиента."
	if got := StripCitationMarkers(raw); got != raw {
		t.Fatalf("StripCitationMarkers() = %q, want prose preserved", got)
	}
}

func TestStripCitationMarkersFooterOnly(t *testing.T) {
	sourceID := "article:22222222-2222-2222-2222-222222222222:1"
	raw := "Ответ по базе.\nиспользованные источники: [" + sourceID + "]"
	got := StripCitationMarkers(raw)
	if got != "Ответ по базе." {
		t.Fatalf("StripCitationMarkers() = %q", got)
	}
}
