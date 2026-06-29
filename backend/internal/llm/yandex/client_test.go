package yandex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/knowledgeos/backend/internal/llm"
)

func TestClientChatSendsHeadersPayloadAndParsesResponse(t *testing.T) {
	var seen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer iam-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("x-data-logging-enabled"); got != "false" {
			t.Fatalf("x-data-logging-enabled = %q", got)
		}
		if got := r.Header.Get("x-folder-id"); got != "folder" {
			t.Fatalf("x-folder-id = %q", got)
		}
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt://folder/yandexgpt-lite/latest" {
			t.Fatalf("model = %q", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Fatalf("messages = %+v", req.Messages)
		}
		if len(req.Tools) != 1 || req.Tools[0].Function.Name != "search_knowledge" {
			t.Fatalf("tools = %+v", req.Tools)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"model":"gpt://folder/yandexgpt-lite/latest",
			"choices":[{"message":{"role":"assistant","content":"answer","tool_calls":[{"id":"call-1","type":"function","function":{"name":"search_knowledge","arguments":{"q":"hello"}}}]}}],
			"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), Config{
		Endpoint:         server.URL,
		FolderID:         "folder",
		AuthToken:        "iam-token",
		DefaultChatModel: "gpt://folder/yandexgpt-lite/latest",
	})
	resp, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
		Tools: []llm.Tool{{
			Name:       "search_knowledge",
			Parameters: json.RawMessage(`{"type":"object"}`),
		}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if !seen {
		t.Fatal("server was not called")
	}
	if resp.Message.Content != "answer" || resp.Usage.TotalTokens != 7 {
		t.Fatalf("response = %+v", resp)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Name != "search_knowledge" {
		t.Fatalf("tool calls = %+v", resp.Message.ToolCalls)
	}
}

func TestClientChatAPIKeyAuthAndValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Api-Key api-key" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), Config{
		Endpoint:         server.URL,
		AuthToken:        "api-key",
		AuthType:         authTypeAPIKey,
		DefaultChatModel: "model",
	})
	if _, err := client.Chat(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}}); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if _, err := client.Chat(context.Background(), llm.ChatRequest{}); err == nil || !strings.Contains(err.Error(), "messages") {
		t.Fatalf("Chat() empty messages error = %v", err)
	}
	noModel := NewClient(server.Client(), Config{Endpoint: server.URL})
	if _, err := noModel.Chat(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}}); err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("Chat() no model error = %v", err)
	}
}

func TestClientRetriesOnlyRetryableStatuses(t *testing.T) {
	var retryableCalls int32
	retryable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&retryableCalls, 1)
		if call == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer retryable.Close()
	client := NewClient(retryable.Client(), Config{Endpoint: retryable.URL, DefaultChatModel: "model", MaxRetries: 1})
	if _, err := client.Chat(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}}); err != nil {
		t.Fatalf("retryable Chat() error = %v", err)
	}
	if retryableCalls != 2 {
		t.Fatalf("retryable calls = %d, want 2", retryableCalls)
	}

	var nonRetryableCalls int32
	nonRetryable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&nonRetryableCalls, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer nonRetryable.Close()
	client = NewClient(nonRetryable.Client(), Config{Endpoint: nonRetryable.URL, DefaultChatModel: "model", MaxRetries: 2})
	if _, err := client.Chat(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}}); err == nil {
		t.Fatal("Chat() expected error")
	}
	if nonRetryableCalls != 1 {
		t.Fatalf("non retryable calls = %d, want 1", nonRetryableCalls)
	}
}

func TestClientChatStreamReadsSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.Client(), Config{Endpoint: server.URL, DefaultChatModel: "model"})
	ch, err := client.ChatStream(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	var got string
	var done bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error = %v", chunk.Err)
		}
		got += chunk.Delta
		done = done || chunk.Done
	}
	if got != "hello" || !done {
		t.Fatalf("stream got %q done=%v", got, done)
	}
}

func TestClientChatStreamReportsDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {bad-json}\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.Client(), Config{Endpoint: server.URL, DefaultChatModel: "model"})
	ch, err := client.ChatStream(context.Background(), llm.ChatRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	chunk := <-ch
	if chunk.Err == nil {
		t.Fatal("expected stream decode error")
	}
}

func TestClientEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if r.URL.Path != "/embeddings" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		switch req.Model {
		case "doc-model":
			if len(req.Input) != 1 {
				t.Fatalf("doc input = %+v", req.Input)
			}
			switch req.Input[0] {
			case "a":
				w.Write([]byte(`{"data":[{"index":0,"embedding":[0.1,0.2]}]}`))
			case "b":
				w.Write([]byte(`{"data":[{"index":0,"embedding":[0.2,0.3]}]}`))
			default:
				t.Fatalf("unexpected doc input = %q", req.Input[0])
			}
		case "query-model":
			w.Write([]byte(`{"data":[{"index":0,"embedding":[0.9,0.8]}]}`))
		default:
			t.Fatalf("unexpected model = %q", req.Model)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), Config{
		Endpoint:            server.URL,
		EmbeddingDocModel:   "doc-model",
		EmbeddingQueryModel: "query-model",
	})
	docs, err := client.EmbedDocs(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("EmbedDocs() error = %v", err)
	}
	if len(docs) != 2 || docs[0][0] != 0.1 || docs[1][0] != 0.2 {
		t.Fatalf("docs = %+v", docs)
	}
	query, err := client.EmbedQuery(context.Background(), "q")
	if err != nil {
		t.Fatalf("EmbedQuery() error = %v", err)
	}
	if len(query) != 2 || query[0] != 0.9 {
		t.Fatalf("query = %+v", query)
	}
}

func TestClientEmbeddingValidation(t *testing.T) {
	client := NewClient(nil, Config{})
	if _, err := client.EmbedDocs(context.Background(), []string{"a"}); err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("EmbedDocs() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"index":2,"embedding":[0.1]}]}`))
	}))
	defer server.Close()
	client = NewClient(server.Client(), Config{Endpoint: server.URL, EmbeddingDocModel: "doc-model", Timeout: time.Second})
	if _, err := client.EmbedDocs(context.Background(), []string{"a"}); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("EmbedDocs() out-of-range error = %v", err)
	}
}
