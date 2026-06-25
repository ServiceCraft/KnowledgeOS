package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/knowledgeos/backend/internal/logger"
)

type Config struct {
	// Postgres
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	// App
	AppProfile         string // "local" or "cloud"
	JWTSecret          string
	SuperadminEmail    string
	SuperadminPassword string
	LogLevel           string
	LogFormat          string

	// Sync
	CloudAPIURL         string
	CloudAPIKey         string
	SyncIntervalSeconds int

	// Backup snapshot endpoint
	BackupCodePath   string // path to the source tree included in code.tar.gz
	BackupGitCommit  string // current commit hash, surfaced in metadata.json
	BackupCommitFile string // optional file to read the commit hash from when env is empty

	// Bot / LLM
	SecretsEncryptionKey       string
	YandexEndpoint             string
	YandexFolderID             string
	YandexAPIKey               string
	YandexDefaultChatModelLite string
	YandexDefaultChatModelPro  string
	YandexEmbeddingDocModel    string
	YandexEmbeddingQueryModel  string
	YandexTimeoutSeconds       int
	YandexMaxRetries           int

	// RAG
	RAGWorkerEnabled             bool
	RAGWorkerBatchSize           int
	RAGWorkerPollIntervalSeconds int
	RAGIndexMaxAttempts          int
	RAGVectorTopK                int
	RAGHybridTopK                int
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.PostgresDB,
	)
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("POSTGRES_PORT", "5432"))
	syncInterval, _ := strconv.Atoi(getEnv("SYNC_INTERVAL_SECONDS", "60"))
	yandexTimeout, _ := strconv.Atoi(getEnv("YANDEX_TIMEOUT_SECONDS", "30"))
	yandexRetries, _ := strconv.Atoi(getEnv("YANDEX_MAX_RETRIES", "3"))
	ragWorkerBatchSize, _ := strconv.Atoi(getEnv("RAG_WORKER_BATCH_SIZE", "10"))
	ragWorkerPollInterval, _ := strconv.Atoi(getEnv("RAG_WORKER_POLL_INTERVAL_SECONDS", "5"))
	ragIndexMaxAttempts, _ := strconv.Atoi(getEnv("RAG_INDEX_MAX_ATTEMPTS", "5"))
	ragVectorTopK, _ := strconv.Atoi(getEnv("RAG_VECTOR_TOP_K", "20"))
	ragHybridTopK, _ := strconv.Atoi(getEnv("RAG_HYBRID_TOP_K", "8"))

	return &Config{
		PostgresHost:                 getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:                 port,
		PostgresUser:                 getEnv("POSTGRES_USER", "knowledgeos"),
		PostgresPassword:             getEnv("POSTGRES_PASSWORD", "changeme"),
		PostgresDB:                   getEnv("POSTGRES_DB", "knowledgeos"),
		AppProfile:                   getEnv("APP_PROFILE", "local"),
		JWTSecret:                    getEnv("JWT_SECRET", ""),
		SuperadminEmail:              getEnv("SUPERADMIN_EMAIL", ""),
		SuperadminPassword:           getEnv("SUPERADMIN_PASSWORD", ""),
		LogLevel:                     getEnv("LOG_LEVEL", logger.DefaultLevel()),
		LogFormat:                    getEnv("LOG_FORMAT", logger.DefaultFormat()),
		CloudAPIURL:                  getEnv("CLOUD_API_URL", ""),
		CloudAPIKey:                  getEnv("CLOUD_API_KEY", ""),
		SyncIntervalSeconds:          syncInterval,
		BackupCodePath:               getEnv("BACKUP_CODE_PATH", "/app/src"),
		BackupGitCommit:              getEnv("BACKUP_GIT_COMMIT", ""),
		BackupCommitFile:             getEnv("BACKUP_COMMIT_FILE", "/app/COMMIT"),
		SecretsEncryptionKey:         getEnv("SECRETS_ENCRYPTION_KEY", ""),
		YandexEndpoint:               getEnv("YANDEX_ENDPOINT", "https://ai.api.cloud.yandex.net/v1"),
		YandexFolderID:               getEnv("YANDEX_FOLDER_ID", ""),
		YandexAPIKey:                 getEnv("YANDEX_API_KEY", ""),
		YandexDefaultChatModelLite:   getEnv("YANDEX_DEFAULT_CHAT_MODEL_LITE", ""),
		YandexDefaultChatModelPro:    getEnv("YANDEX_DEFAULT_CHAT_MODEL_PRO", ""),
		YandexEmbeddingDocModel:      getEnv("YANDEX_EMBEDDING_DOC_MODEL", ""),
		YandexEmbeddingQueryModel:    getEnv("YANDEX_EMBEDDING_QUERY_MODEL", ""),
		YandexTimeoutSeconds:         yandexTimeout,
		YandexMaxRetries:             yandexRetries,
		RAGWorkerEnabled:             getEnv("RAG_WORKER_ENABLED", "true") == "true",
		RAGWorkerBatchSize:           ragWorkerBatchSize,
		RAGWorkerPollIntervalSeconds: ragWorkerPollInterval,
		RAGIndexMaxAttempts:          ragIndexMaxAttempts,
		RAGVectorTopK:                ragVectorTopK,
		RAGHybridTopK:                ragHybridTopK,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
