package main

import (
	"context"
	"log"

	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/database"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
	syncagent "github.com/knowledgeos/backend/internal/sync"
)

func main() {
	cfg := config.Load()

	if cfg.CloudAPIURL == "" || cfg.CloudAPIKey == "" {
		log.Fatal("CLOUD_API_URL and CLOUD_API_KEY must be set for sync agent")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
		log.Fatal("No companies found to sync")
	}

	agent := syncagent.NewAgent(cfg, syncSvc, companies[0].ID)

	log.Println("KnowledgeOS Sync Agent starting...")
	agent.Run()
}
