package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/knowledgeos/backend/internal/channels"
)

func TestRegisterWebhookCallsSetWebhook(t *testing.T) {
	var calledURL string
	var body string
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calledURL = req.URL.String()
		data, _ := io.ReadAll(req.Body)
		body = string(data)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: http.Header{}}, nil
	})})

	registered, err := adapter.RegisterWebhook(
		t.Context(),
		channels.ChannelConfig{Token: "token", Metadata: json.RawMessage(`{"webhook_secret":"secret"}`)},
		"https://example.com/api/v1/webhooks/telegram/company",
	)
	if err != nil {
		t.Fatalf("RegisterWebhook returned error: %v", err)
	}
	if !registered {
		t.Fatal("expected webhook to be registered")
	}
	if !strings.Contains(calledURL, "/bottoken/setWebhook") {
		t.Fatalf("unexpected request URL: %s", calledURL)
	}
	if !strings.Contains(body, `"secret_token":"secret"`) || !strings.Contains(body, `"url":"https://example.com/api/v1/webhooks/telegram/company"`) {
		t.Fatalf("unexpected request body: %s", body)
	}
}

func TestCheckWebhookCallsGetWebhookInfo(t *testing.T) {
	var calledURL string
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calledURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{"url":"https://example.com/api/v1/webhooks/telegram/company"}}`)),
			Header:     http.Header{},
		}, nil
	})})

	status, err := adapter.CheckWebhook(
		t.Context(),
		channels.ChannelConfig{Token: "token"},
		"https://example.com/api/v1/webhooks/telegram/company",
	)
	if err != nil {
		t.Fatalf("CheckWebhook returned error: %v", err)
	}
	if !status.Configured || status.Error != "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if !strings.Contains(calledURL, "/bottoken/getWebhookInfo") {
		t.Fatalf("unexpected request URL: %s", calledURL)
	}
}

func TestCheckWebhookMismatchDoesNotExposeURLs(t *testing.T) {
	adapter := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{"url":"https://other.example.com/hook"}}`)),
			Header:     http.Header{},
		}, nil
	})})

	status, err := adapter.CheckWebhook(
		t.Context(),
		channels.ChannelConfig{Token: "token"},
		"https://example.com/api/v1/webhooks/telegram/company",
	)
	if err != nil {
		t.Fatalf("CheckWebhook returned error: %v", err)
	}
	if status.Configured {
		t.Fatal("expected webhook to be reported as not configured")
	}
	if status.Error == "" || strings.Contains(status.Error, "https://") {
		t.Fatalf("unexpected error text: %q", status.Error)
	}
}

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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
