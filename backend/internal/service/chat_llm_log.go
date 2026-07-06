package service

import (
	"context"
	"encoding/json"
	"fmt"

	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/llm"
	"github.com/knowledgeos/backend/internal/privacy"
)

// maxDebugLogField bounds the size of any single debug field emitted by the
// explicitly-opt-in BOT_CHAT_DEBUG_LOG mode.
const maxDebugLogField = 2000

// safeDebugField redacts PII and truncates a value for the debug log mode.
func safeDebugField(s string) string {
	s = privacy.Redact(s)
	if len(s) > maxDebugLogField {
		return s[:maxDebugLogField] + "...(truncated)"
	}
	return s
}

func (s *ChatService) logLLMExchange(
	ctx context.Context,
	companyID, sessionID uuid.UUID,
	phase string,
	messages []llm.Message,
	response string,
	toolCalls []llm.ToolCall,
) {
	if !s.debugLog {
		return
	}

	event := applog.From(ctx).Info().
		Str("company_id", companyID.String()).
		Str("session_id", sessionID.String()).
		Str("chat_llm_phase", phase)

	if len(messages) > 0 {
		if messages[0].Role == llm.RoleSystem {
			event = event.Str("system_prompt", safeDebugField(messages[0].Content))
		}
		if encoded, err := json.Marshal(messages); err == nil {
			event = event.Str("llm_messages", safeDebugField(string(encoded)))
		}
	}
	if response != "" {
		event = event.Str("llm_response", safeDebugField(response))
	}
	if len(toolCalls) > 0 {
		names := make([]string, 0, len(toolCalls))
		for _, call := range toolCalls {
			names = append(names, call.Name)
		}
		event = event.Strs("llm_tool_calls", names)
	}
	event.Msg("chat llm exchange")
}

func chatLLMPhaseToolLoop(iter int) string {
	return fmt.Sprintf("tool_loop_%d", iter)
}

func (s *ChatService) logChatSkippedLLM(ctx context.Context, companyID, sessionID uuid.UUID, reason string, sourceCount int) {
	if !s.debugLog {
		return
	}
	applog.From(ctx).Info().
		Str("company_id", companyID.String()).
		Str("session_id", sessionID.String()).
		Str("skip_reason", reason).
		Int("source_count", sourceCount).
		Msg("chat llm skipped")
}
