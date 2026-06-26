package eval

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/knowledgeos/backend/internal/domain"
)

func msg(action domain.GuardrailAction, reason string, withSource bool, cited []string) *domain.ChatMessage {
	m := &domain.ChatMessage{GuardrailAction: action, RefusalReason: reason}
	if withSource {
		m.Sources, _ = json.Marshal([]domain.ChatSource{{SourceID: "qa:x:0", Title: "src"}})
	}
	m.CitedSourceIDs, _ = json.Marshal(cited)
	return m
}

func TestDefaultCasesLoad(t *testing.T) {
	cases, err := DefaultCases()
	if err != nil {
		t.Fatalf("DefaultCases() error = %v", err)
	}
	if len(cases) == 0 {
		t.Fatalf("no fixtures loaded")
	}
	var in, out int
	for _, c := range cases {
		switch c.Kind {
		case KindInDomain:
			in++
		case KindOutOfDomain:
			out++
		default:
			t.Fatalf("case %s has invalid kind %q", c.ID, c.Kind)
		}
	}
	if in == 0 || out == 0 {
		t.Fatalf("expected both in-domain and out-of-domain fixtures, got in=%d out=%d", in, out)
	}
}

func TestRunPerfectBotScoresFull(t *testing.T) {
	cases, _ := DefaultCases()
	kindByQ := map[string]Kind{}
	for _, c := range cases {
		kindByQ[c.Question] = c.Kind
	}
	answer := func(_ context.Context, q string) (*domain.ChatMessage, error) {
		if kindByQ[q] == KindInDomain {
			return msg(domain.GuardrailActionAnswer, "", true, []string{"qa:x:0"}), nil
		}
		return msg(domain.GuardrailActionRefuse, "no_context", false, nil), nil
	}
	rep := Run(context.Background(), cases, answer)
	if rep.Passed != rep.Total {
		t.Fatalf("passed %d/%d, want all", rep.Passed, rep.Total)
	}
	if rep.GroundingRate != 1 || rep.RefusalAccuracy != 1 {
		t.Fatalf("metrics grounding=%v refusal=%v, want 1/1", rep.GroundingRate, rep.RefusalAccuracy)
	}
}

func TestRunDetectsRegressions(t *testing.T) {
	cases := []Case{
		{ID: "a", Question: "in", Kind: KindInDomain},
		{ID: "b", Question: "bad-in", Kind: KindInDomain},
		{ID: "c", Question: "out", Kind: KindOutOfDomain},
		{ID: "d", Question: "leaky-out", Kind: KindOutOfDomain},
	}
	answer := func(_ context.Context, q string) (*domain.ChatMessage, error) {
		switch q {
		case "in":
			return msg(domain.GuardrailActionAnswer, "", true, []string{"qa:x:0"}), nil
		case "bad-in": // answered but without any source -> not grounded
			return msg(domain.GuardrailActionAnswer, "", false, nil), nil
		case "out":
			return msg(domain.GuardrailActionEscalate, "low_confidence", false, nil), nil
		default: // leaky-out answered instead of refusing
			return msg(domain.GuardrailActionAnswer, "", false, nil), nil
		}
	}
	rep := Run(context.Background(), cases, answer)
	if rep.GroundingRate != 0.5 {
		t.Fatalf("grounding rate = %v, want 0.5", rep.GroundingRate)
	}
	if rep.RefusalAccuracy != 0.5 {
		t.Fatalf("refusal accuracy = %v, want 0.5", rep.RefusalAccuracy)
	}
	if rep.Passed != 2 {
		t.Fatalf("passed = %d, want 2", rep.Passed)
	}
}

func TestReportRendersMarkdownAndJSON(t *testing.T) {
	cases, _ := DefaultCases()
	answer := func(_ context.Context, _ string) (*domain.ChatMessage, error) {
		return msg(domain.GuardrailActionRefuse, "no_context", false, nil), nil
	}
	rep := Run(context.Background(), cases, answer)
	if md := rep.Markdown(); len(md) == 0 || md[0] != '#' {
		t.Fatalf("markdown report malformed")
	}
	if _, err := rep.JSON(); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
}
