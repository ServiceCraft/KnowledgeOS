package config

import (
	"fmt"
	"os"
	"strconv"
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

	// Sync
	CloudAPIURL         string
	CloudAPIKey         string
	SyncIntervalSeconds int

	// Backup snapshot endpoint
	BackupCodePath   string // path to the source tree included in code.tar.gz
	BackupGitCommit  string // current commit hash, surfaced in metadata.json
	BackupCommitFile string // optional file to read the commit hash from when env is empty
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

	return &Config{
		PostgresHost:        getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:        port,
		PostgresUser:        getEnv("POSTGRES_USER", "knowledgeos"),
		PostgresPassword:    getEnv("POSTGRES_PASSWORD", "changeme"),
		PostgresDB:          getEnv("POSTGRES_DB", "knowledgeos"),
		AppProfile:          getEnv("APP_PROFILE", "local"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		SuperadminEmail:     getEnv("SUPERADMIN_EMAIL", ""),
		SuperadminPassword:  getEnv("SUPERADMIN_PASSWORD", ""),
		CloudAPIURL:         getEnv("CLOUD_API_URL", ""),
		CloudAPIKey:         getEnv("CLOUD_API_KEY", ""),
		SyncIntervalSeconds: syncInterval,
		BackupCodePath:      getEnv("BACKUP_CODE_PATH", "/app/src"),
		BackupGitCommit:     getEnv("BACKUP_GIT_COMMIT", ""),
		BackupCommitFile:    getEnv("BACKUP_COMMIT_FILE", "/app/COMMIT"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
