package max

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

func TestRegisterWebhookUsesDefaultMAXAPI(t *testing.T) {
	var calledURL, authHeader, body string
	const testMAXAPI = "https://example.com/subscriptions"
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calledURL = req.URL.String()
		authHeader = req.Header.Get("Authorization")
		data, _ := io.ReadAll(req.Body)
		body = string(data)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
	})})

	meta := json.RawMessage(`{"webhook_secret":"hook-secret","webhook_registration_url":"` + testMAXAPI + `"}`)
	registered, err := adapter.RegisterWebhook(
		context.Background(),
		channels.ChannelConfig{Token: "max-token", Metadata: meta},
		"https://public.example.com/api/v1/webhooks/max/company",
	)
	if err != nil {
		t.Fatalf("RegisterWebhook returned error: %v", err)
	}
	if !registered {
		t.Fatal("expected webhook to be registered")
	}
	if calledURL != testMAXAPI {
		t.Fatalf("unexpected registration URL: %s", calledURL)
	}
	if authHeader != "max-token" {
		t.Fatalf("expected raw token in Authorization, got %q", authHeader)
	}
	if strings.Contains(authHeader, "Bearer") {
		t.Fatalf("MAX API must not use Bearer prefix, got %q", authHeader)
	}
	if !strings.Contains(body, `"secret":"hook-secret"`) || !strings.Contains(body, `"url":"https://public.example.com/api/v1/webhooks/max/company"`) {
		t.Fatalf("unexpected request body: %s", body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

func TestCheckWebhookFindsRegisteredURLAmongMultiple(t *testing.T) {
	const testMAXAPI = "https://example.com/subscriptions"
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{"subscriptions":[
				{"url":"https://other.example.com/hook"},
				{"url":"https://public.example.com/api/v1/webhooks/max/company"}
			]}`)),
			Header: http.Header{},
		}, nil
	})})

	status, err := adapter.CheckWebhook(
		context.Background(),
		channels.ChannelConfig{
			Token:    "max-token",
			Metadata: json.RawMessage(`{"subscriptions_url":"` + testMAXAPI + `"}`),
		},
		"https://public.example.com/api/v1/webhooks/max/company",
	)
	if err != nil {
		t.Fatalf("CheckWebhook returned error: %v", err)
	}
	if !status.Configured || status.Error != "" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestCheckWebhookFindsRegisteredURL(t *testing.T) {
	const testMAXAPI = "https://example.com/subscriptions"
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"subscriptions":[{"url":"https://public.example.com/api/v1/webhooks/max/company"}]}`)),
			Header:     http.Header{},
		}, nil
	})})

	status, err := adapter.CheckWebhook(
		context.Background(),
		channels.ChannelConfig{
			Token:    "max-token",
			Metadata: json.RawMessage(`{"subscriptions_url":"` + testMAXAPI + `"}`),
		},
		"https://public.example.com/api/v1/webhooks/max/company",
	)
	if err != nil {
		t.Fatalf("CheckWebhook returned error: %v", err)
	}
	if !status.Configured || status.Error != "" {
		t.Fatalf("unexpected status: %#v", status)
	}
}
