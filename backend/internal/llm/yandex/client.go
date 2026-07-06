package yandex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/knowledgeos/backend/internal/llm"
	applog "github.com/knowledgeos/backend/internal/logger"
)

const (
	defaultEndpoint      = "https://ai.api.cloud.yandex.net/v1"
	authTypeIAM          = "iam"
	authTypeAPIKey       = "api_key"
	maxLoggedBodyBytes   = 8192
	maxResponseBodyBytes = 16 << 20 // 16 MiB safety cap when buffering non-stream responses
)

type Config struct {
	Endpoint            string
	FolderID            string
	AuthToken           string
	AuthType            string
	DefaultChatModel    string
	EmbeddingDocModel   string
	EmbeddingQueryModel string
	Timeout             time.Duration
	MaxRetries          int
}

type Client struct {
	httpClient *http.Client
	cfg        Config
}

// NewClient executes the yandex.NewClient operation.
func NewClient(httpClient *http.Client, cfg Config) *Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	if cfg.AuthType == "" {
		cfg.AuthType = authTypeIAM
	}
	return &Client{httpClient: httpClient, cfg: cfg}
}

// Chat executes the yandex.Client.Chat operation.
func (c *Client) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	applog.TraceCall(ctx, "yandex.Client.Chat")
	if err := c.validateChatRequest(req); err != nil {
		return nil, err
	}
	applog.From(ctx).Debug().
		Int("messages", len(req.Messages)).
		Int("tools", len(req.Tools)).
		Str("model", firstNonEmpty(req.Model, c.cfg.DefaultChatModel)).
		Msg("yandex chat request started")
	body, err := json.Marshal(c.toChatRequest(req, false))
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, "/chat/completions", body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("chat response has no choices")
	}
	applog.From(ctx).Debug().
		Str("model", out.Model).
		Int("prompt_tokens", out.Usage.PromptTokens).
		Int("completion_tokens", out.Usage.CompletionTokens).
		Msg("yandex chat request completed")
	return &llm.ChatResponse{
		Message: llm.ToolMessage{
			Role:      llm.Role(out.Choices[0].Message.Role),
			Content:   out.Choices[0].Message.Content,
			ToolCalls: fromOpenAIToolCalls(out.Choices[0].Message.ToolCalls),
		},
		Usage: fromOpenAIUsage(out.Usage),
		Model: out.Model,
	}, nil
}

// ChatStream executes the yandex.Client.ChatStream operation.
func (c *Client) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	applog.TraceCall(ctx, "yandex.Client.ChatStream")
	if err := c.validateChatRequest(req); err != nil {
		return nil, err
	}
	applog.From(ctx).Debug().
		Int("messages", len(req.Messages)).
		Int("tools", len(req.Tools)).
		Str("model", firstNonEmpty(req.Model, c.cfg.DefaultChatModel)).
		Msg("yandex stream chat request started")
	body, err := json.Marshal(c.toChatRequest(req, true))
	if err != nil {
		return nil, fmt.Errorf("marshal stream chat request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, "/chat/completions", body, true)
	if err != nil {
		return nil, err
	}

	ch := make(chan llm.StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				applog.From(ctx).Debug().Msg("yandex stream chat completed")
				ch <- llm.StreamChunk{Done: true}
				return
			}
			var event openAIStreamResponse
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				ch <- llm.StreamChunk{Err: fmt.Errorf("decode stream chunk: %w", err)}
				return
			}
			if len(event.Choices) == 0 {
				continue
			}
			ch <- llm.StreamChunk{
				Delta:     event.Choices[0].Delta.Content,
				ToolCalls: fromOpenAIToolCalls(event.Choices[0].Delta.ToolCalls),
				Usage:     fromOpenAIUsage(event.Usage),
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- llm.StreamChunk{Err: fmt.Errorf("read stream: %w", err)}
		}
	}()
	return ch, nil
}

// EmbedDocs executes the yandex.Client.EmbedDocs operation.
func (c *Client) EmbedDocs(ctx context.Context, texts []string) ([][]float32, error) {
	applog.TraceCall(ctx, "yandex.Client.EmbedDocs")
	return c.embed(ctx, c.cfg.EmbeddingDocModel, texts)
}

// EmbedQuery executes the yandex.Client.EmbedQuery operation.
func (c *Client) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	applog.TraceCall(ctx, "yandex.Client.EmbedQuery")
	vectors, err := c.embed(ctx, c.cfg.EmbeddingQueryModel, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("embedding response has no vectors")
	}
	return vectors[0], nil
}

func (c *Client) embed(ctx context.Context, model string, texts []string) ([][]float32, error) {
	applog.TraceCall(ctx, "yandex.Client.embed")
	if model == "" {
		return nil, errors.New("embedding model is required")
	}
	if len(texts) == 0 {
		return nil, nil
	}
	applog.From(ctx).Debug().
		Str("model", model).
		Int("texts", len(texts)).
		Msg("yandex embeddings request started")

	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vector, err := c.embedOne(ctx, model, text)
		if err != nil {
			return nil, err
		}
		vectors[i] = vector
	}
	applog.From(ctx).Debug().
		Str("model", model).
		Int("vectors", len(vectors)).
		Msg("yandex embeddings request completed")
	return vectors, nil
}

func (c *Client) embedOne(ctx context.Context, model, text string) ([]float32, error) {
	req := openAIEmbeddingRequest{
		Model: model,
		Input: []string{text},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, "/embeddings", body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(out.Data) == 0 {
		return nil, errors.New("embedding response has no vectors")
	}
	for _, item := range out.Data {
		if item.Index != 0 {
			continue
		}
		if len(item.Embedding) == 0 {
			return nil, errors.New("embedding response has empty vector")
		}
		return item.Embedding, nil
	}
	if out.Data[0].Index != 0 {
		return nil, fmt.Errorf("embedding index out of range: %d", out.Data[0].Index)
	}
	return nil, errors.New("embedding response has no vectors")
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, stream bool) (*http.Response, error) {
	applog.TraceCall(ctx, "yandex.Client.do")
	attempts := c.cfg.MaxRetries + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := sleepBackoff(ctx, attempt); err != nil {
				return nil, err
			}
		}
		// Never log request bodies: they contain prompts, RAG context and user
		// messages. Only structural metadata is safe to emit.
		applog.From(ctx).Debug().
			Str("method", method).
			Str("path", path).
			Int("attempt", attempt+1).
			Int("body_size", len(body)).
			Msg("yandex http request")

		httpReq, err := http.NewRequestWithContext(ctx, method, c.cfg.Endpoint+path, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create yandex request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		if stream {
			httpReq.Header.Set("Accept", "text/event-stream")
		}
		httpReq.Header.Set("x-data-logging-enabled", "false")
		if c.cfg.FolderID != "" {
			httpReq.Header.Set("x-folder-id", c.cfg.FolderID)
		}
		if c.cfg.AuthToken != "" {
			switch c.cfg.AuthType {
			case authTypeAPIKey:
				httpReq.Header.Set("Authorization", "Api-Key "+c.cfg.AuthToken)
			default:
				httpReq.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
			}
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("yandex request: %w", err)
			applog.From(ctx).Debug().
				Err(err).
				Str("method", method).
				Str("path", path).
				Int("attempt", attempt+1).
				Msg("yandex http request failed")
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if stream {
				applog.From(ctx).Debug().
					Str("method", method).
					Str("path", path).
					Int("status", resp.StatusCode).
					Int("attempt", attempt+1).
					Msg("yandex http request succeeded")
				return resp, nil
			}
			respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read yandex response: %w", readErr)
			}
			// Response bodies contain generated content; log only the size.
			applog.From(ctx).Debug().
				Str("method", method).
				Str("path", path).
				Int("status", resp.StatusCode).
				Int("attempt", attempt+1).
				Int("body_size", len(respBody)).
				Msg("yandex http request succeeded")
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			return resp, nil
		}

		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxLoggedBodyBytes+1))
		_ = resp.Body.Close()
		respLog := truncateForLog(respBody)
		lastErr = yandexHTTPError(resp.StatusCode, respLog)
		log := applog.From(ctx)
		event := log.Debug()
		if !retryableStatus(resp.StatusCode) {
			event = log.Error().Err(lastErr)
		}
		// Do not log request/response bodies here either; the upstream error text
		// is carried in lastErr for server-side diagnostics only.
		event.
			Str("method", method).
			Str("path", path).
			Int("status", resp.StatusCode).
			Int("attempt", attempt+1).
			Bool("retryable", retryableStatus(resp.StatusCode)).
			Int("body_size", len(body)).
			Msg("yandex http request returned non-success status")
		if !retryableStatus(resp.StatusCode) {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func truncateForLog(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) > maxLoggedBodyBytes {
		return string(body[:maxLoggedBodyBytes]) + "...(truncated)"
	}
	return string(body)
}

func yandexHTTPError(status int, responseBody string) error {
	responseBody = strings.TrimSpace(responseBody)
	if responseBody == "" {
		return fmt.Errorf("yandex request failed with status %d", status)
	}
	return fmt.Errorf("yandex request failed with status %d: %s", status, responseBody)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (c *Client) validateChatRequest(req llm.ChatRequest) error {
	if len(req.Messages) == 0 {
		return errors.New("messages are required")
	}
	if req.Model == "" && c.cfg.DefaultChatModel == "" {
		return errors.New("chat model is required")
	}
	return nil
}

func (c *Client) toChatRequest(req llm.ChatRequest, stream bool) openAIChatRequest {
	model := req.Model
	if model == "" {
		model = c.cfg.DefaultChatModel
	}
	out := openAIChatRequest{
		Model:       model,
		Messages:    make([]openAIMessage, 0, len(req.Messages)),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      stream,
	}
	for _, msg := range req.Messages {
		out.Messages = append(out.Messages, openAIMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toOpenAIToolCalls(msg.ToolCalls),
		})
	}
	for _, tool := range req.Tools {
		out.Tools = append(out.Tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return out
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func sleepBackoff(ctx context.Context, attempt int) error {
	applog.TraceCall(ctx, "yandex.sleepBackoff")
	delay := time.Duration(100*(1<<min(attempt-1, 5))) * time.Millisecond
	delay += time.Duration(rand.Intn(100)) * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
