package channels

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionsContainURL(t *testing.T) {
	expected := "https://bot.example.com/api/v1/webhooks/max/company-id"
	raw := json.RawMessage(`{"subscriptions":[{"url":"https://bot.example.com/api/v1/webhooks/max/company-id/","update_types":["message_created"]}]}`)
	if !SubscriptionsContainURL(raw, expected) {
		t.Fatal("expected subscription url match")
	}
	if SubscriptionsContainURL(raw, "https://other.example.com/hook") {
		t.Fatal("expected different url to not match")
	}
}

func TestSubscriptionsContainURLTelegramResult(t *testing.T) {
	raw := json.RawMessage(`{"ok":true,"result":{"url":"https://bot.example.com/api/v1/webhooks/telegram/company-id","has_custom_certificate":false}}`)
	if !SubscriptionsContainURL(raw, "https://bot.example.com/api/v1/webhooks/telegram/company-id") {
		t.Fatal("expected telegram-style payload match")
	}
}

func TestSubscriptionsContainURLMatchesAmongMultiple(t *testing.T) {
	expected := "https://bot.example.com/api/v1/webhooks/max/company-id"
	raw := json.RawMessage(`{"subscriptions":[
		{"url":"https://other.example.com/hook"},
		{"webhook_url":"https://bot.example.com/api/v1/webhooks/max/company-id/"}
	]}`)
	if !SubscriptionsContainURL(raw, expected) {
		t.Fatal("expected match among multiple subscriptions")
	}
}

func TestExtractSubscriptionURLsCollectsAll(t *testing.T) {
	raw := json.RawMessage(`{"webhooks":[
		{"url":"https://a.example.com/hook"},
		{"url":"https://b.example.com/hook"}
	]}`)
	urls := ExtractSubscriptionURLs(raw)
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d: %v", len(urls), urls)
	}
}
