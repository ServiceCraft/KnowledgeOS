package tools

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/llm"
	applog "github.com/knowledgeos/backend/internal/logger"
)

// ValidationError signals that a tool call was rejected before execution, either
// because the tool is unknown or because its arguments failed schema validation.
// The chat loop feeds these back to the model as a tool result instead of
// failing the turn, so the model can recover.
type ValidationError struct {
	Msg string
}

// Error executes the tools.ValidationError.Error operation.
func (e *ValidationError) Error() string { return e.Msg }

// Execute validates and runs a single tool call. Unknown tools and arguments
// that fail JSON-schema validation are rejected with a *ValidationError before
// the underlying tool is ever invoked (Б4 acceptance criterion).
func (r *Registry) Execute(ctx context.Context, companyID uuid.UUID, call llm.ToolCall) (Result, error) {
	applog.TraceCall(ctx, "tools.Registry.Execute")
	tool, ok := r.Get(call.Name)
	if !ok {
		return Result{}, &ValidationError{Msg: fmt.Sprintf("unknown tool: %s", call.Name)}
	}
	if err := validateArguments(tool.Schema(), call.Arguments); err != nil {
		return Result{}, err
	}
	return tool.Execute(ctx, companyID, call.Arguments)
}
