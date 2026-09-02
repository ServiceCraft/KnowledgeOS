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
	// CallbackData/CallbackID are set when the update is an inline-button press
	// (Telegram callback_query) rather than a text message. CallbackID is used to
	// acknowledge the press so the client's button stops showing a spinner.
	CallbackData string
	CallbackID   string
}

// MessageButton is one inline button. Data is the callback payload sent back
// when the button is pressed.
type MessageButton struct {
	Text string
	Data string
}

type OutboundMessage struct {
	ExternalChatID string
	Text           string
	Sources        []domain.ChatSource
	// Buttons renders as an inline keyboard on channels that support it
	// (Telegram). Channels without button support ignore this and send Text only.
	Buttons []MessageButton
}

// CallbackAnswerer is implemented by adapters that can acknowledge an inline
// button press (Telegram answerCallbackQuery). Optional — type-asserted.
type CallbackAnswerer interface {
	AnswerCallback(ctx context.Context, cfg ChannelConfig, callbackID, text string) error
}

type WebhookStatus struct {
	Configured bool
	Error      string
}

type WebhookChecker interface {
	CheckWebhook(ctx context.Context, cfg ChannelConfig, webhookURL string) (WebhookStatus, error)
}

// WebhookSubscriptionsLister returns the raw subscription/webhook payload the
// upstream provider reports for the bot token. It is used to let admins inspect
// which webhooks are currently registered for a channel.
type WebhookSubscriptionsLister interface {
	ListSubscriptions(ctx context.Context, cfg ChannelConfig) (json.RawMessage, error)
}

type Adapter interface {
	Channel() domain.ChatChannel
	SecretKind() domain.SecretKind
	RegisterWebhook(ctx context.Context, cfg ChannelConfig, webhookURL string) (bool, error)
	ParseInbound(r WebhookRequest, cfg ChannelConfig) (*InboundMessage, *WebhookResponse, error)
	SendTyping(ctx context.Context, cfg ChannelConfig, externalChatID string) error
	SendMessage(ctx context.Context, cfg ChannelConfig, msg OutboundMessage) error
}
