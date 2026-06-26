package telegram

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
)

func TestParseInboundVerifiesSecretAndMapsMessage(t *testing.T) {
	adapter := New(nil)
	req := channels.WebhookRequest{
		Headers: http.Header{"X-Telegram-Bot-Api-Secret-Token": []string{"secret"}},
		Body:    []byte(`{"update_id":42,"message":{"text":"hello","chat":{"id":123}}}`),
	}
	msg, immediate, err := adapter.ParseInbound(req, channels.ChannelConfig{Metadata: json.RawMessage(`{"webhook_secret":"secret"}`)})
	if err != nil {
		t.Fatalf("ParseInbound returned error: %v", err)
	}
	if immediate != nil {
		t.Fatalf("unexpected immediate response: %#v", immediate)
	}
	if msg.ExternalChatID != "123" || msg.Text != "hello" || msg.UpdateID != "42" {
		t.Fatalf("unexpected message: %#v", msg)
	}
}

func TestParseInboundRejectsInvalidSecret(t *testing.T) {
	adapter := New(nil)
	_, _, err := adapter.ParseInbound(channels.WebhookRequest{
		Headers: http.Header{"X-Telegram-Bot-Api-Secret-Token": []string{"wrong"}},
		Body:    []byte(`{}`),
	}, channels.ChannelConfig{Metadata: json.RawMessage(`{"webhook_secret":"secret"}`)})
	if err == nil {
		t.Fatal("expected invalid secret error")
	}
}
