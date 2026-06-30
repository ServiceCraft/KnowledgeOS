package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knowledgeos/backend/internal/config"
	secretcrypto "github.com/knowledgeos/backend/internal/crypto"
	"github.com/knowledgeos/backend/internal/database"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	applog.Configure(cfg.LogLevel, cfg.LogFormat, "knowledgeos-rag-worker")

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	s := store.New(db, cfg.AppProfile)
	articleStore := store.NewArticleStore(s)
	qaStore := store.NewQAStore(s)
	pricingStore := store.NewPricingStore(s)
	kbEmbeddingStore := store.NewKBEmbeddingStore(s)
	kbIndexJobStore := store.NewKBIndexJobStore(s)
	botSettingsStore := store.NewBotSettingsStore(s)
	tenantSecretStore := store.NewTenantSecretStore(s)

	secretCipher, err := secretcrypto.NewCipher(cfg.SecretsEncryptionKey)
	if err != nil {
		log.Warn().Err(err).Msg("bot tenant secrets are disabled")
	}
	botSettingsSvc := service.NewBotSettingsService(botSettingsStore)
	tenantSecretSvc := service.NewTenantSecretService(tenantSecretStore, secretCipher)
	llmFactory := service.NewLLMFactory(cfg, botSettingsSvc, tenantSecretSvc)

	ragIndexerSvc := service.NewRAGIndexerService(
		kbIndexJobStore,
		kbEmbeddingStore,
		articleStore,
		qaStore,
		pricingStore,
		llmFactory,
		service.RAGIndexerConfig{
			WorkerEnabled:      true,
			WorkerBatchSize:    cfg.RAGWorkerBatchSize,
			WorkerPollInterval: time.Duration(cfg.RAGWorkerPollIntervalSeconds) * time.Second,
			IndexMaxAttempts:   cfg.RAGIndexMaxAttempts,
		},
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ragIndexerSvc.StartWorker(ctx)
	log.Info().Msg("rag worker running")
	<-ctx.Done()
	log.Info().Msg("rag worker stopped")
}
