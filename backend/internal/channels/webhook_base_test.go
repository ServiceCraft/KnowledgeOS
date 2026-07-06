package channels

import (
	"encoding/json"
	"testing"

	"github.com/knowledgeos/backend/internal/domain"
)

func TestWebhookDisplayBaseUsesRequestHostWhenPublicHTTPSUnknown(t *testing.T) {
	got := WebhookDisplayBase("http://staging.example.com:8081", "")
	if got != "http://staging.example.com:8081" {
		t.Fatalf("WebhookDisplayBase() = %q, want request host", got)
	}
}

func TestWebhookDisplayBasePrefersConfiguredPublicHTTPS(t *testing.T) {
	got := WebhookDisplayBase("http://staging.example.com:8081", "https://bot.example.com")
	if got != "https://bot.example.com" {
		t.Fatalf("WebhookDisplayBase() = %q, want configured public base", got)
	}
}

func TestChannelRequiredFieldsComplete(t *testing.T) {
	telegramMeta := json.RawMessage(`{"webhook_secret":"********"}`)
	if !ChannelRequiredFieldsComplete(domain.ChatChannelTelegram, telegramMeta) {
		t.Fatal("expected telegram with webhook_secret to be complete")
	}
	if ChannelRequiredFieldsComplete(domain.ChatChannelTelegram, json.RawMessage(`{}`)) {
		t.Fatal("expected empty metadata to be incomplete")
	}
	if ChannelRequiredFieldsComplete(domain.ChatChannelMAX, json.RawMessage(`{"handoff_notification_chat_id":"1"}`)) {
		t.Fatal("expected max without webhook_secret to be incomplete")
	}
}
