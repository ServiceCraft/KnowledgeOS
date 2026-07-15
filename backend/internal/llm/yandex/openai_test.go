package yandex

import (
	"encoding/json"
	"testing"

	"github.com/knowledgeos/backend/internal/llm"
)

func TestUnquoteToolArgumentsWireString(t *testing.T) {
	// The OpenAI wire format delivers function.arguments as a JSON string.
	got := unquoteToolArguments(json.RawMessage(`"{\"staff_id\": 7, \"date\": \"2026-07-15\"}"`))
	var args map[string]any
	if err := json.Unmarshal(got, &args); err != nil {
		t.Fatalf("unquoted arguments are not a JSON object: %v (%s)", err, got)
	}
	if args["date"] != "2026-07-15" {
		t.Fatalf("arguments = %v", args)
	}
}

func TestUnquoteToolArgumentsPassthroughAndEmpty(t *testing.T) {
	obj := json.RawMessage(`{"a":1}`)
	if got := unquoteToolArguments(obj); string(got) != `{"a":1}` {
		t.Fatalf("raw object must pass through, got %s", got)
	}
	if got := unquoteToolArguments(json.RawMessage(`""`)); string(got) != `{}` {
		t.Fatalf("empty string arguments must become {}, got %s", got)
	}
	if got := unquoteToolArguments(nil); got != nil {
		t.Fatalf("nil must pass through, got %s", got)
	}
}

func TestQuoteToolArgumentsRoundTrip(t *testing.T) {
	calls := []llm.ToolCall{{ID: "c1", Name: "yclients_get_times", Arguments: json.RawMessage(`{"staff_id":7}`)}}
	wire := toOpenAIToolCalls(calls)
	if string(wire[0].Function.Arguments) != `"{\"staff_id\":7}"` {
		t.Fatalf("history arguments must be a JSON string, got %s", wire[0].Function.Arguments)
	}
	back := fromOpenAIToolCalls(wire)
	if string(back[0].Arguments) != `{"staff_id":7}` {
		t.Fatalf("round trip mismatch: %s", back[0].Arguments)
	}
	// Already-quoted arguments must not be double-quoted.
	quoted := quoteToolArguments(json.RawMessage(`"{}"`))
	if string(quoted) != `"{}"` {
		t.Fatalf("double quoting: %s", quoted)
	}
}
