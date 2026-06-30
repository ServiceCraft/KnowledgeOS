package config

import (
	"context"
	"fmt"
	"time"

	"github.com/knowledgeos/backend/internal/logger"
	"github.com/sethvargo/go-envconfig"
)

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
	return &cfg
}
