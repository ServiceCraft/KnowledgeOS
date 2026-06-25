package main

import (
	"context"

	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/database"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
	syncagent "github.com/knowledgeos/backend/internal/sync"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	applog.Configure(cfg.LogLevel, cfg.LogFormat, "knowledgeos-sync")

	if cfg.CloudAPIURL == "" || cfg.CloudAPIKey == "" {
		log.Fatal().Msg("CLOUD_API_URL and CLOUD_API_KEY must be set for sync agent")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	s := store.New(db, cfg.AppProfile)

	themeStore := store.NewThemeStore(s)
	qaStore := store.NewQAStore(s)
	pricingStore := store.NewPricingStore(s)
	articleStore := store.NewArticleStore(s)
	commentStore := store.NewCommentStore(s)
	linkStore := store.NewLinkStore(s)
	syncStore := store.NewSyncStore(s)

	syncSvc := service.NewSyncService(syncStore, themeStore, qaStore, pricingStore, articleStore, commentStore, linkStore)

	// Find the company to sync (first company in DB)
	companyStore := store.NewCompanyStore(s)
	companies, _, err := companyStore.List(context.Background(), domain.CompanyFilter{Page: 1, Limit: 1})
	if err != nil || len(companies) == 0 {
		log.Fatal().Err(err).Msg("no companies found to sync")
	}

	agent := syncagent.NewAgent(cfg, syncSvc, companies[0].ID)

	log.Info().Str("company_id", companies[0].ID.String()).Msg("knowledgeos sync agent starting")
	agent.Run()
}
