// Command chat-eval runs the Б5 guardrail eval harness against a running
// KnowledgeOS API: it logs in, opens a chat session, sends each fixture question
// and scores the guardrail verdicts, writing a JSON + Markdown report. The
// report is the acceptance artifact for in-domain and provocative question sets.
//
// Usage:
//
//	go run ./cmd/chat-eval -base http://localhost:8080/api/v1 \
//	  -email admin@example.com -password changeme -out ../docs/eval
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/knowledgeos/backend/internal/chat/eval"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
)

func main() {
	base := flag.String("base", "http://localhost:8080/api/v1", "API base URL")
	email := flag.String("email", "admin@example.com", "login email")
	password := flag.String("password", "changeme", "login password")
	out := flag.String("out", "docs/eval", "output directory for the report")
	flag.Parse()

	if err := run(*base, *email, *password, *out); err != nil {
		fmt.Fprintln(os.Stderr, "chat-eval:", err)
		os.Exit(1)
	}
}

func run(base, email, password, outDir string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	ctx := context.Background()

	token, err := login(ctx, client, base, email, password)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	sessionID, err := createSession(ctx, client, base, token)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	cases, err := eval.DefaultCases()
	if err != nil {
		return fmt.Errorf("load fixtures: %w", err)
	}

	answer := func(ctx context.Context, question string) (*domain.ChatMessage, error) {
		return sendMessage(ctx, client, base, token, sessionID, question)
	}
	report := eval.Run(ctx, cases, answer)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return applog.TraceErr(ctx, "chat eval: create output dir failed", err)
	}
	jsonBytes, err := report.JSON()
	if err != nil {
		return applog.TraceErr(ctx, "chat eval: marshal report failed", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "last-run.json"), jsonBytes, 0o644); err != nil {
		return applog.TraceErr(ctx, "chat eval: write json report failed", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "last-run.md"), []byte(report.Markdown()), 0o644); err != nil {
		return applog.TraceErr(ctx, "chat eval: write markdown report failed", err)
	}

	fmt.Printf("eval complete: %d/%d passed | grounding %.0f%% | refusal %.0f%%\n",
		report.Passed, report.Total, report.GroundingRate*100, report.RefusalAccuracy*100)
	fmt.Printf("report written to %s\n", outDir)
	return nil
}

func login(ctx context.Context, c *http.Client, base, email, password string) (string, error) {
	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	body := map[string]string{"email": email, "password": password}
	if err := doJSON(ctx, c, http.MethodPost, base+"/auth/login", "", body, &resp); err != nil {
		return "", err
	}
	if resp.Data.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}
	return resp.Data.AccessToken, nil
}

func createSession(ctx context.Context, c *http.Client, base, token string) (string, error) {
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	body := map[string]string{"channel": "playground", "title": "guardrail eval"}
	if err := doJSON(ctx, c, http.MethodPost, base+"/admin/bot/chat/sessions", token, body, &resp); err != nil {
		return "", err
	}
	if resp.Data.ID == "" {
		return "", fmt.Errorf("empty session id")
	}
	return resp.Data.ID, nil
}

func sendMessage(ctx context.Context, c *http.Client, base, token, sessionID, question string) (*domain.ChatMessage, error) {
	var resp struct {
		Data struct {
			Message domain.ChatMessage `json:"message"`
		} `json:"data"`
	}
	url := fmt.Sprintf("%s/admin/bot/chat/sessions/%s/messages", base, sessionID)
	body := map[string]string{"content": question}
	if err := doJSON(ctx, c, http.MethodPost, url, token, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Data.Message, nil
}

func doJSON(ctx context.Context, c *http.Client, method, url, token string, body, out interface{}) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return applog.TraceErr(ctx, "chat eval: marshal request failed", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return applog.TraceErr(ctx, "chat eval: build request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.Do(req)
	if err != nil {
		return applog.TraceErr(ctx, "chat eval: http request failed", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return applog.TraceErr(ctx, "chat eval: read response failed", err)
	}
	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s: status %d: %s", method, url, res.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}
