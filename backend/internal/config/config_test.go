package config

import "testing"

func TestLoadBotYandexDefaults(t *testing.T) {
	t.Setenv("YANDEX_TIMEOUT_SECONDS", "30")
	t.Setenv("YANDEX_MAX_RETRIES", "3")

	cfg := Load()
	if cfg.LogLevel != "info" || cfg.LogFormat != "json" {
		t.Fatalf("log defaults = %q/%q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.YandexEndpoint != "https://ai.api.cloud.yandex.net/v1" {
		t.Fatalf("YandexEndpoint = %q", cfg.YandexEndpoint)
	}
	if cfg.YandexTimeoutSeconds != 30 {
		t.Fatalf("YandexTimeoutSeconds = %d", cfg.YandexTimeoutSeconds)
	}
	if cfg.YandexMaxRetries != 3 {
		t.Fatalf("YandexMaxRetries = %d", cfg.YandexMaxRetries)
	}
	if cfg.YandexEmbeddingDocModel != "" || cfg.YandexEmbeddingQueryModel != "" {
		t.Fatalf("embedding defaults = %q/%q, want empty env defaults", cfg.YandexEmbeddingDocModel, cfg.YandexEmbeddingQueryModel)
	}
	if cfg.BotChatDebugLog {
		t.Fatalf("BotChatDebugLog = true, want false by default")
	}
}

func TestLoadBotYandexCustomValues(t *testing.T) {
	t.Setenv("SECRETS_ENCRYPTION_KEY", "secret-key")
	t.Setenv("YANDEX_ENDPOINT", "https://example.test/v1")
	t.Setenv("YANDEX_FOLDER_ID", "folder")
	t.Setenv("YANDEX_API_KEY", "api-key")
	t.Setenv("YANDEX_DEFAULT_CHAT_MODEL_LITE", "lite")
	t.Setenv("YANDEX_DEFAULT_CHAT_MODEL_PRO", "pro")
	t.Setenv("YANDEX_EMBEDDING_DOC_MODEL", "doc")
	t.Setenv("YANDEX_EMBEDDING_QUERY_MODEL", "query")
	t.Setenv("YANDEX_TIMEOUT_SECONDS", "7")
	t.Setenv("YANDEX_MAX_RETRIES", "5")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "console")
	t.Setenv("BOT_CHAT_DEBUG_LOG", "true")

	cfg := Load()
	if cfg.SecretsEncryptionKey != "secret-key" ||
		cfg.LogLevel != "debug" ||
		cfg.LogFormat != "console" ||
		cfg.BotChatDebugLog != true ||
		cfg.YandexEndpoint != "https://example.test/v1" ||
		cfg.YandexFolderID != "folder" ||
		cfg.YandexAPIKey != "api-key" ||
		cfg.YandexDefaultChatModelLite != "lite" ||
		cfg.YandexDefaultChatModelPro != "pro" ||
		cfg.YandexEmbeddingDocModel != "doc" ||
		cfg.YandexEmbeddingQueryModel != "query" ||
		cfg.YandexTimeoutSeconds != 7 ||
		cfg.YandexMaxRetries != 5 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
