package yandex

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/knowledgeos/backend/internal/llm"
)

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta openAIMessage `json:"delta"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func toOpenAIToolCalls(calls []llm.ToolCall) []openAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]openAIToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, openAIToolCall{
			ID:   call.ID,
			Type: "function",
			Function: openAIToolFunction{
				Name:      call.Name,
				Arguments: quoteToolArguments(call.Arguments),
			},
		})
	}
	return out
}

// quoteToolArguments encodes tool-call arguments as the JSON string the OpenAI
// wire format expects. Internally arguments are stored as a raw JSON object, so
// they must be re-wrapped before being echoed back in conversation history.
func quoteToolArguments(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}
	if trimmed[0] == '"' {
		return trimmed
	}
	quoted, err := json.Marshal(string(trimmed))
	if err != nil {
		return trimmed
	}
	return quoted
}

// unquoteToolArguments unwraps the OpenAI wire format, where function.arguments
// is a JSON-encoded string ("{\"a\":1}"), into the raw JSON object the tool
// framework validates against. Raw objects pass through unchanged.
func unquoteToolArguments(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return raw
	}
	var inner string
	if err := json.Unmarshal(trimmed, &inner); err != nil {
		return raw
	}
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(inner)
}

func fromOpenAIToolCalls(calls []openAIToolCall) []llm.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]llm.ToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, llm.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: unquoteToolArguments(call.Function.Arguments),
		})
	}
	return out
}

func fromOpenAIUsage(usage openAIUsage) llm.Usage {
	return llm.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}
