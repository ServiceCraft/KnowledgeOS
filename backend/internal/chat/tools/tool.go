// Package tools implements the Б4 tool-calling framework: a registry of
// whitelisted tools exposed to the LLM, JSON-schema validation of arguments and
// a dispatcher that executes a single tool call against tenant-scoped services.
package tools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

// Result is the outcome of a single tool execution. Content is the textual
// payload handed back to the LLM as a tool message; Sources are the knowledge
// base citations discovered while executing the tool, so the final answer can
// be grounded.
type Result struct {
	Content string
	Sources []domain.ChatSource
}

// Tool is a single capability the bot can invoke. Name and Schema are advertised
// to the LLM; Execute runs the tool after its arguments have been validated.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error)
}
