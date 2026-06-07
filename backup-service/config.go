package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the backup-service settings, all sourced from environment
// variables (TZ §5.6).
type Config struct {
	AppURL        string
	AppAPIKey     string
	GitHubRepoURL string
	GitHubToken   string
	GitHubBranch  string
	BackupsPath   string
	BackupsKeep   int
	ScheduleCron  string
	HTTPTimeout   time.Duration

	// Operational paths/identity (sensible defaults, overridable).
	RepoWorkdir    string
	GitAuthorName  string
	GitAuthorEmail string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		AppURL:         os.Getenv("APP_URL"),
		AppAPIKey:      os.Getenv("APP_API_KEY"),
		GitHubRepoURL:  os.Getenv("GITHUB_REPO_URL"),
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		GitHubBranch:   getEnv("GITHUB_BRANCH", "backup"),
		BackupsPath:    getEnv("BACKUPS_PATH", "/data/backups"),
		ScheduleCron:   getEnv("SCHEDULE_CRON", "0 3 * * *"),
		RepoWorkdir:    getEnv("REPO_WORKDIR", "/data/repo"),
		GitAuthorName:  getEnv("GIT_AUTHOR_NAME", "KnowledgeOS Backup"),
		GitAuthorEmail: getEnv("GIT_AUTHOR_EMAIL", "backup@knowledgeos.local"),
	}

	keep, err := strconv.Atoi(getEnv("BACKUPS_KEEP", "30"))
	if err != nil || keep < 1 {
		return cfg, fmt.Errorf("invalid BACKUPS_KEEP: %q", os.Getenv("BACKUPS_KEEP"))
	}
	cfg.BackupsKeep = keep

	timeoutSec, err := strconv.Atoi(getEnv("HTTP_TIMEOUT_SECONDS", "600"))
	if err != nil || timeoutSec < 1 {
		return cfg, fmt.Errorf("invalid HTTP_TIMEOUT_SECONDS: %q", os.Getenv("HTTP_TIMEOUT_SECONDS"))
	}
	cfg.HTTPTimeout = time.Duration(timeoutSec) * time.Second

	if cfg.AppURL == "" {
		return cfg, fmt.Errorf("APP_URL is required")
	}
	if cfg.AppAPIKey == "" {
		return cfg, fmt.Errorf("APP_API_KEY is required")
	}
	if cfg.GitHubRepoURL == "" {
		return cfg, fmt.Errorf("GITHUB_REPO_URL is required")
	}
	if cfg.GitHubToken == "" {
		return cfg, fmt.Errorf("GITHUB_TOKEN is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
