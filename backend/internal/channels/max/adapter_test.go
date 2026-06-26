package max

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
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
