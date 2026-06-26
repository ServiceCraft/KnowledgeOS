package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ChatChannel string

const (
	ChatChannelPlayground ChatChannel = "playground"
	ChatChannelAPI        ChatChannel = "api"
	ChatChannelTelegram   ChatChannel = "telegram"
	ChatChannelMAX        ChatChannel = "max"
	ChatChannelVK         ChatChannel = "vk"
)

type ChatState string

const (
	ChatStateBot             ChatState = "bot"
	ChatStateWaitingOperator ChatState = "waiting_operator"
	ChatStateOperator        ChatState = "operator"
	ChatStateClosed          ChatState = "closed"
)

// GuardrailAction is the machine-readable outcome of the Б5 guardrail layer for
// an assistant turn: a normal grounded answer, a polite refusal or an
// escalation signal for the future operator handoff module.
type GuardrailAction string

const (
	GuardrailActionAnswer   GuardrailAction = "answer"
	GuardrailActionRefuse   GuardrailAction = "refuse"
	GuardrailActionEscalate GuardrailAction = "escalate"
)

type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleTool      ChatRole = "tool"
	ChatRoleOperator  ChatRole = "operator"
)

type ChatSession struct {
	BaseModel
	CompanyID      uuid.UUID   `gorm:"type:uuid;not null;index" json:"company_id"`
	Channel        ChatChannel `gorm:"type:text;not null;default:playground" json:"channel"`
	ExternalChatID *string     `gorm:"type:text" json:"external_chat_id,omitempty"`
	State          ChatState   `gorm:"type:text;not null;default:bot" json:"state"`
	OperatorID     *uuid.UUID  `gorm:"type:uuid" json:"operator_id,omitempty"`
	Title          string      `gorm:"type:text;not null;default:''" json:"title"`
	LastMessageAt  *time.Time  `json:"last_message_at,omitempty"`
}

// TableName executes the domain.ChatSession.TableName operation.
func (ChatSession) TableName() string { return "chat_sessions" }

type ChatMessage struct {
	BaseModel
	CompanyID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"company_id"`
	SessionID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"session_id"`
	Role             ChatRole        `gorm:"type:text;not null" json:"role"`
	Content          string          `gorm:"type:text;not null;default:''" json:"content"`
	ToolCallID       string          `gorm:"type:text;not null;default:''" json:"tool_call_id,omitempty"`
	ToolCalls        json.RawMessage `gorm:"type:jsonb;not null;default:'[]'" json:"tool_calls"`
	Sources          json.RawMessage `gorm:"type:jsonb;not null;default:'[]'" json:"sources"`
	GuardrailAction  GuardrailAction `gorm:"type:text;not null;default:answer" json:"guardrail_action,omitempty"`
	ConfidenceScore  *float64        `gorm:"type:numeric(4,3)" json:"confidence_score,omitempty"`
	RefusalReason    string          `gorm:"type:text;not null;default:''" json:"refusal_reason,omitempty"`
	CitedSourceIDs   json.RawMessage `gorm:"type:jsonb;not null;default:'[]'" json:"cited_source_ids,omitempty"`
	TokensPrompt     int             `gorm:"not null;default:0" json:"tokens_prompt"`
	TokensCompletion int             `gorm:"not null;default:0" json:"tokens_completion"`
}

// TableName executes the domain.ChatMessage.TableName operation.
func (ChatMessage) TableName() string { return "chat_messages" }

// SourcesList decodes the stored Sources JSON into a slice of ChatSource.
func (m *ChatMessage) SourcesList() []ChatSource {
	if len(m.Sources) == 0 {
		return nil
	}
	var out []ChatSource
	if err := json.Unmarshal(m.Sources, &out); err != nil {
		return nil
	}
	return out
}

type ChatSource struct {
	SourceID   string       `json:"source_id"`
	EntityType KBEntityType `json:"entity_type"`
	EntityID   uuid.UUID    `json:"entity_id"`
	ChunkIdx   int          `json:"chunk_idx"`
	Title      string       `json:"title"`
	Content    string       `json:"content"`
	Snippet    string       `json:"snippet,omitempty"`
	Score      float64      `json:"score"`
}

type ChatSessionFilter struct {
	Page  int
	Limit int
}

type ChatSessionWithMessages struct {
	Session  *ChatSession  `json:"session"`
	Messages []ChatMessage `json:"messages"`
}

type ChatRepository interface {
	CreateSession(ctx context.Context, companyID uuid.UUID, session *ChatSession) error
	ListSessions(ctx context.Context, companyID uuid.UUID, filter ChatSessionFilter) ([]ChatSession, int64, error)
	GetSession(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*ChatSession, error)
	UpdateSession(ctx context.Context, companyID uuid.UUID, session *ChatSession) error
	AppendMessage(ctx context.Context, companyID uuid.UUID, message *ChatMessage) error
	ListMessages(ctx context.Context, companyID uuid.UUID, sessionID uuid.UUID, limit int) ([]ChatMessage, error)
}
