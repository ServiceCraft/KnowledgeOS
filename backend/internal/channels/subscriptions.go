package channels

import (
	"encoding/json"
	"strings"
)

var subscriptionURLFields = []string{
	"url",
	"webhook_url",
	"webhookUrl",
	"endpoint",
	"callback_url",
	"callbackUrl",
}

var subscriptionContainerKeys = []string{
	"subscriptions",
	"webhooks",
	"items",
	"data",
	"result",
}

// ExtractSubscriptionURLs returns every webhook URL found in a provider
// subscriptions/webhook payload. Trailing slashes are stripped.
func ExtractSubscriptionURLs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	urls := make([]string, 0)
	add := func(url string) {
		url = normalizeWebhookCompareURL(url)
		if url == "" {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		urls = append(urls, url)
	}
	for _, item := range extractSubscriptionRecords(raw) {
		add(subscriptionRecordURL(item))
	}
	var top map[string]interface{}
	if json.Unmarshal(raw, &top) == nil {
		add(stringField(top, "url"))
	}
	return urls
}

// SubscriptionsContainURL reports whether provider subscription payload includes
// the expected public webhook URL (case-sensitive path, trailing slash ignored).
func SubscriptionsContainURL(raw json.RawMessage, expectedURL string) bool {
	expectedURL = normalizeWebhookCompareURL(expectedURL)
	if expectedURL == "" {
		return false
	}
	for _, url := range ExtractSubscriptionURLs(raw) {
		if url == expectedURL {
			return true
		}
	}
	return false
}

func normalizeWebhookCompareURL(url string) string {
	return strings.TrimRight(strings.TrimSpace(url), "/")
}

func extractSubscriptionRecords(raw json.RawMessage) []map[string]interface{} {
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	return extractSubscriptionRecordsFromValue(data)
}

func extractSubscriptionRecordsFromValue(data interface{}) []map[string]interface{} {
	switch value := data.(type) {
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(value))
		for _, item := range value {
			if record, ok := item.(map[string]interface{}); ok {
				out = append(out, record)
			}
		}
		return out
	case map[string]interface{}:
		for _, key := range subscriptionContainerKeys {
			if nested, ok := value[key]; ok {
				if records := extractSubscriptionRecordsFromValue(nested); len(records) > 0 {
					return records
				}
			}
		}
		return []map[string]interface{}{value}
	default:
		return nil
	}
}

func subscriptionRecordURL(item map[string]interface{}) string {
	for _, key := range subscriptionURLFields {
		if url := stringField(item, key); url != "" {
			return normalizeWebhookCompareURL(url)
		}
	}
	return ""
}

func stringField(item map[string]interface{}, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
