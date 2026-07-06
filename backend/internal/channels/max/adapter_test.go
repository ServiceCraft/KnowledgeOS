package max

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/service"
)

func TestParseInboundVerifiesSecretAndMapsMessage(t *testing.T) {
	adapter := New(nil)
	req := channels.WebhookRequest{
		Headers: http.Header{"X-Max-Bot-Api-Secret": []string{"secret"}},
		Body:    []byte(`{"update_id":"u1","message":{"text":"hello","chat_id":"chat-1"}}`),
	}
	msg, immediate, err := adapter.ParseInbound(req, channels.ChannelConfig{Metadata: json.RawMessage(`{"webhook_secret":"secret"}`)})
	if err != nil {
		t.Fatalf("ParseInbound returned error: %v", err)
	}
	if immediate != nil {
		t.Fatalf("unexpected immediate response: %#v", immediate)
	}
	if msg.ExternalChatID != "chat-1" || msg.Text != "hello" || msg.UpdateID != "u1" {
		t.Fatalf("unexpected message: %#v", msg)
	}
}

func TestSendMessageBlocksUnsafeEndpoints(t *testing.T) {
	adapter := New(nil)
	blocked := []string{
		"http://botapi.max.ru",
		"https://127.0.0.1/messages",
		"https://localhost/messages",
		"https://169.254.169.254/messages",
	}
	for _, endpoint := range blocked {
		cfg := channels.ChannelConfig{
			Token:    "token",
			Metadata: json.RawMessage(`{"send_url":"` + endpoint + `"}`),
		}
		err := adapter.SendMessage(context.Background(), cfg, channels.OutboundMessage{ExternalChatID: "chat-1", Text: "hi"})
		if err == nil {
			t.Fatalf("expected blocked endpoint %q to error", endpoint)
		}
		if service.HTTPStatus(err) != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d", endpoint, service.HTTPStatus(err))
		}
	}
}

func TestRegisterWebhookBlocksUnsafeURL(t *testing.T) {
	adapter := New(nil)
	cfg := channels.ChannelConfig{
		Token:    "token",
		Metadata: json.RawMessage(`{"webhook_registration_url":"https://127.0.0.1/register","webhook_secret":"s"}`),
	}
	ok, err := adapter.RegisterWebhook(context.Background(), cfg, "https://public.example.com/webhook")
	if ok || err == nil {
		t.Fatalf("expected blocked registration url to error, got ok=%v err=%v", ok, err)
	}
}
