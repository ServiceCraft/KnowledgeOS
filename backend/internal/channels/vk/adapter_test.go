package vk

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/service"
)

func TestParseInboundConfirmation(t *testing.T) {
	adapter := New(nil)
	_, immediate, err := adapter.ParseInbound(channels.WebhookRequest{
		Body: []byte(`{"type":"confirmation"}`),
	}, channels.ChannelConfig{Metadata: json.RawMessage(`{"confirmation_token":"confirm"}`)})
	if err != nil {
		t.Fatalf("ParseInbound returned error: %v", err)
	}
	if immediate == nil || string(immediate.Body) != "confirm" {
		t.Fatalf("unexpected confirmation response: %#v", immediate)
	}
}

func TestParseInboundMessageNew(t *testing.T) {
	adapter := New(nil)
	msg, immediate, err := adapter.ParseInbound(channels.WebhookRequest{
		Body: []byte(`{"type":"message_new","secret":"secret","object":{"message":{"id":7,"peer_id":123,"text":"hello"}}}`),
	}, channels.ChannelConfig{Metadata: json.RawMessage(`{"secret":"secret"}`)})
	if err != nil {
		t.Fatalf("ParseInbound returned error: %v", err)
	}
	if immediate != nil {
		t.Fatalf("unexpected immediate response: %#v", immediate)
	}
	if msg.ExternalChatID != "123" || msg.Text != "hello" || msg.UpdateID != "7" {
		t.Fatalf("unexpected message: %#v", msg)
	}
}

func TestParseInboundRejectsInvalidSecret(t *testing.T) {
	adapter := New(nil)
	_, _, err := adapter.ParseInbound(channels.WebhookRequest{
		Body: []byte(`{"type":"message_new","secret":"wrong"}`),
	}, channels.ChannelConfig{Metadata: json.RawMessage(`{"secret":"secret"}`)})
	if err == nil || service.HTTPStatus(err) != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}
