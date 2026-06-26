package channels

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/knowledgeos/backend/internal/domain"
)

type WebhookRequest struct {
	Headers http.Header
	Body    []byte
	Query   map[string][]string
}

type WebhookResponse struct {
	Status      int
	ContentType string
	Body        []byte
}

type ChannelConfig struct {
	Token    string
	Metadata json.RawMessage
}

type InboundMessage struct {
	Channel        domain.ChatChannel
	ExternalChatID string
	UpdateID       string
	Text           string
}

type OutboundMessage struct {
	ExternalChatID string
	Text           string
	Sources        []domain.ChatSource
}

type Adapter interface {
	Channel() domain.ChatChannel
	SecretKind() domain.SecretKind
	RegisterWebhook(ctx context.Context, cfg ChannelConfig, webhookURL string) (bool, error)
	ParseInbound(r WebhookRequest, cfg ChannelConfig) (*InboundMessage, *WebhookResponse, error)
	SendTyping(ctx context.Context, cfg ChannelConfig, externalChatID string) error
	SendMessage(ctx context.Context, cfg ChannelConfig, msg OutboundMessage) error
}
