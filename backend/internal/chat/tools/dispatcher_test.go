package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type spyTool struct {
	name   string
	schema string
	called int
}

func (t *spyTool) Name() string            { return t.name }
func (t *spyTool) Description() string     { return "spy" }
func (t *spyTool) Schema() json.RawMessage { return json.RawMessage(t.schema) }
func (t *spyTool) Execute(_ context.Context, _ uuid.UUID, _ json.RawMessage) (Result, error) {
	t.called++
	return Result{Content: "ok"}, nil
}

func TestRegistryExecuteValidArgsRunsTool(t *testing.T) {
	tool := &spyTool{name: "search_knowledge", schema: `{"type":"object","properties":{"query":{"type":"string","minLength":1}},"required":["query"]}`}
	reg := NewRegistry(tool)

	res, err := reg.Execute(context.Background(), uuid.New(), toolCall("search_knowledge", `{"query":"hi"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.Content != "ok" {
		t.Fatalf("content = %q", res.Content)
	}
	if tool.called != 1 {
		t.Fatalf("tool called %d times, want 1", tool.called)
	}
}

func TestRegistryExecuteInvalidArgsRejectedBeforeTool(t *testing.T) {
	tool := &spyTool{name: "search_knowledge", schema: `{"type":"object","properties":{"query":{"type":"string","minLength":1}},"required":["query"]}`}
	reg := NewRegistry(tool)

	cases := map[string]string{
		"missing required": `{}`,
		"wrong type":       `{"query":123}`,
		"too short":        `{"query":""}`,
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := reg.Execute(context.Background(), uuid.New(), toolCall("search_knowledge", args))
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("error = %v, want *ValidationError", err)
			}
		})
	}
	if tool.called != 0 {
		t.Fatalf("tool executed %d times despite invalid args", tool.called)
	}
}

func TestRegistryExecuteNullArgsTreatedAsEmptyObject(t *testing.T) {
	tool := &spyTool{name: "no_args", schema: `{"type":"object"}`}
	reg := NewRegistry(tool)

	for _, args := range []string{`null`, ``, `   `, `{}`} {
		t.Run("args="+args, func(t *testing.T) {
			if _, err := reg.Execute(context.Background(), uuid.New(), toolCall("no_args", args)); err != nil {
				t.Fatalf("Execute(%q) error = %v, want nil", args, err)
			}
		})
	}
	if tool.called != 4 {
		t.Fatalf("tool called %d times, want 4", tool.called)
	}
}

func TestRegistryExecuteUnknownToolRejected(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Execute(context.Background(), uuid.New(), toolCall("does_not_exist", `{}`))
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
}

func TestRegistryDefinitionsPreserveOrder(t *testing.T) {
	reg := NewRegistry(
		&spyTool{name: "a", schema: `{"type":"object"}`},
		&spyTool{name: "b", schema: `{"type":"object"}`},
	)
	defs := reg.Definitions()
	if len(defs) != 2 || defs[0].Name != "a" || defs[1].Name != "b" {
		t.Fatalf("definitions = %+v", defs)
	}
}
