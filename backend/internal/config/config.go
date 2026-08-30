package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knowledgeos/backend/internal/logger"
	"github.com/sethvargo/go-envconfig"
)

// minJWTSecretLength is the minimum acceptable JWT secret length (bytes) for
// non-local profiles.
const minJWTSecretLength = 32

type Config struct {
	PostgresHost     string `env:"POSTGRES_HOST, default=localhost"`
	PostgresPort     int    `env:"POSTGRES_PORT, default=5432"`
	PostgresUser     string `env:"POSTGRES_USER, default=knowledgeos"`
	PostgresPassword string `env:"POSTGRES_PASSWORD, default=changeme"`
	PostgresDB       string `env:"POSTGRES_DB, default=knowledgeos"`
	PostgresSSLMode  string `env:"POSTGRES_SSLMODE, default=disable"`
	PGMaxOpenConns   int    `env:"PG_MAX_OPEN_CONNS, default=25"`
	PGMaxIdleConns   int    `env:"PG_MAX_IDLE_CONNS, default=10"`
	PGConnMaxLifeMin int    `env:"PG_CONN_MAX_LIFETIME_MIN, default=30"`

	AppProfile         string `env:"APP_PROFILE, default=local"`
	JWTSecret          string `env:"JWT_SECRET"`
	SuperadminEmail    string `env:"SUPERADMIN_EMAIL"`
	SuperadminPassword string `env:"SUPERADMIN_PASSWORD"`
	LogLevel           string `env:"LOG_LEVEL, default=info"`
	LogFormat          string `env:"LOG_FORMAT, default=json"`

	AuthAccessTokenHours   int `env:"AUTH_ACCESS_TOKEN_HOURS, default=24"`
	AuthRefreshTokenDays   int `env:"AUTH_REFRESH_TOKEN_DAYS, default=30"`
	AuthListCompaniesLimit int `env:"AUTH_LIST_COMPANIES_LIMIT, default=1000"`

	CacheCompanyPositiveMin int `env:"CACHE_COMPANY_POSITIVE_TTL_MIN, default=5"`
	CacheCompanyNegativeSec int `env:"CACHE_COMPANY_NEGATIVE_TTL_SEC, default=30"`

	CloudAPIURL         string `env:"CLOUD_API_URL"`
	CloudAPIKey         string `env:"CLOUD_API_KEY"`
	SyncIntervalSeconds int    `env:"SYNC_INTERVAL_SECONDS, default=60"`

	BackupCodePath   string `env:"BACKUP_CODE_PATH, default=/app/src"`
	BackupGitCommit  string `env:"BACKUP_GIT_COMMIT"`
	BackupCommitFile string `env:"BACKUP_COMMIT_FILE, default=/app/COMMIT"`

	SecretsEncryptionKey       string `env:"SECRETS_ENCRYPTION_KEY"`
	YandexEndpoint             string `env:"YANDEX_ENDPOINT, default=https://ai.api.cloud.yandex.net/v1"`
	YandexFolderID             string `env:"YANDEX_FOLDER_ID"`
	YandexAPIKey               string `env:"YANDEX_API_KEY"`
	YandexDefaultChatModelLite string `env:"YANDEX_DEFAULT_CHAT_MODEL_LITE"`
	YandexDefaultChatModelPro  string `env:"YANDEX_DEFAULT_CHAT_MODEL_PRO"`
	YandexEmbeddingDocModel    string `env:"YANDEX_EMBEDDING_DOC_MODEL"`
	YandexEmbeddingQueryModel  string `env:"YANDEX_EMBEDDING_QUERY_MODEL"`
	YandexTimeoutSeconds       int    `env:"YANDEX_TIMEOUT_SECONDS, default=30"`
	YandexMaxRetries           int    `env:"YANDEX_MAX_RETRIES, default=3"`

	RAGWorkerEnabled             bool `env:"RAG_WORKER_ENABLED, default=true"`
	RAGWorkerBatchSize           int  `env:"RAG_WORKER_BATCH_SIZE, default=10"`
	RAGWorkerPollIntervalSeconds int  `env:"RAG_WORKER_POLL_INTERVAL_SECONDS, default=5"`
	RAGIndexMaxAttempts          int  `env:"RAG_INDEX_MAX_ATTEMPTS, default=5"`
	RAGVectorTopK                int  `env:"RAG_VECTOR_TOP_K, default=20"`
	RAGHybridTopK                int  `env:"RAG_HYBRID_TOP_K, default=8"`

	BotChatDebugLog bool `env:"BOT_CHAT_DEBUG_LOG, default=false"`

	// PublicWebhookBaseURL is the HTTPS origin used when registering channel
	// webhooks, e.g. https://bot.example.com. Required when admin UI is opened
	// over plain HTTP (local dev) or behind a proxy without X-Forwarded-Proto.
	PublicWebhookBaseURL string `env:"PUBLIC_WEBHOOK_BASE_URL"`

	// TelegramPolling switches the Telegram channel from webhook delivery to
	// long-polling (getUpdates). Needed where Telegram cannot reach our webhook —
	// e.g. inbound Telegram traffic is filtered to a RU-hosted VM — since polling
	// runs over the bot's own outbound connection.
	TelegramPolling bool `env:"TELEGRAM_POLLING, default=false"`

	// TelegramAPIBaseURL overrides the Bot API origin (default
	// https://api.telegram.org). Point it at a relay (e.g. a Cloudflare Worker
	// forwarding to Telegram) when the host's network cannot reach Telegram
	// directly, as on RU-hosted VMs where Telegram is filtered.
	TelegramAPIBaseURL string `env:"TELEGRAM_API_BASE_URL"`

	MaxExtraCACertFile string `env:"MAX_EXTRA_CA_CERT_FILE"`
	MaxExtraCACertPEM  string `env:"MAX_EXTRA_CA_CERT_PEM"`
	MaxInsecureTLS     bool   `env:"MAX_INSECURE_SKIP_VERIFY, default=false"`
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.PostgresDB, c.PostgresSSLMode,
	)
}

// AccessTokenTTL returns configured access token lifetime.
func (c *Config) AccessTokenTTL() time.Duration {
	if c.AuthAccessTokenHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.AuthAccessTokenHours) * time.Hour
}

// RefreshTokenTTL returns configured refresh token lifetime.
func (c *Config) RefreshTokenTTL() time.Duration {
	if c.AuthRefreshTokenDays <= 0 {
		return 30 * 24 * time.Hour
	}
	return time.Duration(c.AuthRefreshTokenDays) * 24 * time.Hour
}

// CompanyCachePositiveTTL returns TTL for positive company existence cache entries.
func (c *Config) CompanyCachePositiveTTL() time.Duration {
	if c.CacheCompanyPositiveMin <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.CacheCompanyPositiveMin) * time.Minute
}

// CompanyCacheNegativeTTL returns TTL for negative company existence cache entries.
func (c *Config) CompanyCacheNegativeTTL() time.Duration {
	if c.CacheCompanyNegativeSec <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.CacheCompanyNegativeSec) * time.Second
}

// PGConnMaxLifetime returns configured connection max lifetime.
func (c *Config) PGConnMaxLifetime() time.Duration {
	if c.PGConnMaxLifeMin <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(c.PGConnMaxLifeMin) * time.Minute
}

// Load reads configuration from environment variables.
func Load() *Config {
	ctx := context.Background()
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = logger.DefaultLevel()
	}
	if cfg.LogFormat == "" {
		cfg.LogFormat = logger.DefaultFormat()
	}
	cfg.validateJWTSecret()
	return &cfg
}

// validateJWTSecret enforces a strong JWT secret outside local development. For
// the local profile a missing secret is replaced with an ephemeral dev-only
// value so the app still boots, but a warning is emitted.
func (c *Config) validateJWTSecret() {
	secret := strings.TrimSpace(c.JWTSecret)
	isLocal := strings.EqualFold(strings.TrimSpace(c.AppProfile), "local")
	if !isLocal {
		if len(secret) < minJWTSecretLength {
			panic(fmt.Sprintf(
				"invalid configuration: JWT_SECRET must be set and at least %d characters for profile %q",
				minJWTSecretLength, c.AppProfile,
			))
		}
		return
	}
	if secret == "" {
		raw := make([]byte, minJWTSecretLength)
		if _, err := rand.Read(raw); err != nil {
			panic(fmt.Sprintf("generate dev JWT secret: %v", err))
		}
		c.JWTSecret = hex.EncodeToString(raw)
		fmt.Fprintln(os.Stderr,
			"WARNING: JWT_SECRET is empty; generated an ephemeral dev-only secret. "+
				"Existing tokens will be invalid on restart. Set JWT_SECRET for stable sessions.")
	}
}
