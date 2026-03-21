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
	AppProfile        string // "local" or "cloud"
	JWTSecret         string
	SuperadminEmail   string
	SuperadminPassword string

	// Sync
	CloudAPIURL         string
	CloudAPIKey         string
	SyncIntervalSeconds int
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
