package vk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/service"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubClient(status int, body string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
}

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
		Body: []byte(`{"event_id":"evt-1","type":"message_new","secret":"secret","object":{"message":{"id":7,"peer_id":123,"text":"hello"}}}`),
	}, channels.ChannelConfig{Metadata: json.RawMessage(`{"secret":"secret"}`)})
	if err != nil {
		t.Fatalf("ParseInbound returned error: %v", err)
	}
	if immediate != nil {
		t.Fatalf("unexpected immediate response: %#v", immediate)
	}
	if msg.ExternalChatID != "123" || msg.Text != "hello" || msg.UpdateID != "evt-1" {
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

func TestSendMessageDetectsAPIErrorOn200(t *testing.T) {
	adapter := New(stubClient(http.StatusOK, `{"error":{"error_code":15,"error_msg":"Access denied"}}`))
	err := adapter.SendMessage(context.Background(), channels.ChannelConfig{Token: "t"}, channels.OutboundMessage{
		ExternalChatID: "123",
		Text:           "hi",
	})
	if err == nil {
		t.Fatalf("expected error when VK returns error object on HTTP 200")
	}
	if service.HTTPStatus(err) != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", service.HTTPStatus(err))
	}
}

func TestSendMessageSucceedsOnOKResponse(t *testing.T) {
	adapter := New(stubClient(http.StatusOK, `{"response":123}`))
	err := adapter.SendMessage(context.Background(), channels.ChannelConfig{Token: "t"}, channels.OutboundMessage{
		ExternalChatID: "123",
		Text:           "hi",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRegisterWebhookBlocksUnsafeURL(t *testing.T) {
	adapter := New(nil)
	blocked := []string{
		"http://vk.example.com/register",
		"https://127.0.0.1/register",
		"https://localhost/register",
		"https://169.254.169.254/register",
	}
	for _, registration := range blocked {
		cfg := channels.ChannelConfig{
			Token: "token",
			Metadata: json.RawMessage(`{"webhook_registration_url":"` + registration +
				`","secret":"s","confirmation_token":"c"}`),
		}
		ok, err := adapter.RegisterWebhook(context.Background(), cfg, "https://public.example.com/webhook")
		if ok || err == nil {
			t.Fatalf("expected blocked registration url %q to error, got ok=%v err=%v", registration, ok, err)
		}
		if service.HTTPStatus(err) != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d", registration, service.HTTPStatus(err))
		}
	}
}
