package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	applog "github.com/knowledgeos/backend/internal/logger"
)

const RequestHandoffToolName = "request_handoff"

type RequestHandoffTool struct{}

func NewRequestHandoffTool() *RequestHandoffTool {
	return &RequestHandoffTool{}
}

func (t *RequestHandoffTool) Name() string { return RequestHandoffToolName }

func (t *RequestHandoffTool) Description() string {
	return "Запросить передачу диалога живому оператору, когда клиент явно просит оператора или бот не может безопасно ответить по базе знаний."
}

func (t *RequestHandoffTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"reason": {"type": "string", "minLength": 1, "description": "Короткая машинная причина эскалации"}
		}
	}`)
}

type requestHandoffArgs struct {
	Reason string `json:"reason"`
}

func (t *RequestHandoffTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.RequestHandoffTool.Execute")
	var in requestHandoffArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return Result{}, &ValidationError{Msg: "invalid arguments for request_handoff"}
		}
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "tool_request"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"handoff_requested": true,
		"reason":            reason,
	})
	return Result{Content: string(payload)}, nil
}
