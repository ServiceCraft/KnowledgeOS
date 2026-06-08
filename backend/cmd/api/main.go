package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/database"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/handler"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret)

	s := store.New(db, cfg.AppProfile)

	// Stores
	companyStore := store.NewCompanyStore(s)
	userStore := store.NewUserStore(s)
	themeStore := store.NewThemeStore(s)
	qaStore := store.NewQAStore(s)
	pricingStore := store.NewPricingStore(s)
	articleStore := store.NewArticleStore(s)
	commentStore := store.NewCommentStore(s)
	linkStore := store.NewLinkStore(s)
	searchStore := store.NewSearchStore(s)
	syncStore := store.NewSyncStore(s)
	callStore := store.NewCallStore(s)
	mentionStore := store.NewQACallMentionStore(s)

	// Services
	authSvc := service.NewAuthService(userStore, syncStore, jwtMgr)
	qaSvc := service.NewQAService(qaStore, themeStore)
	themeSvc := service.NewThemeService(themeStore, qaStore)
	pricingSvc := service.NewPricingService(pricingStore)
	articleSvc := service.NewArticleService(articleStore)
	commentSvc := service.NewCommentService(commentStore, qaStore, articleStore, pricingStore)
	linkSvc := service.NewLinkService(linkStore, qaStore, articleStore, pricingStore)
	searchSvc := service.NewSearchService(searchStore)
	exportSvc := service.NewExportService(db, themeStore, qaStore, pricingStore, articleStore, commentStore, linkStore, callStore, mentionStore)
	syncSvc := service.NewSyncService(syncStore, themeStore, qaStore, pricingStore, articleStore, commentStore, linkStore)
	callSvc := service.NewCallService(callStore, mentionStore, qaStore)
	adminSvc := service.NewAdminService(companyStore, userStore, syncStore)
	userSvc := service.NewUserService(userStore, syncStore)
	snapshotSvc := service.NewSnapshotService(service.SnapshotConfig{
		PGHost:     cfg.PostgresHost,
		PGPort:     cfg.PostgresPort,
		PGUser:     cfg.PostgresUser,
		PGPassword: cfg.PostgresPassword,
		PGDatabase: cfg.PostgresDB,
		CodePath:   cfg.BackupCodePath,
		GitCommit:  cfg.BackupGitCommit,
		CommitFile: cfg.BackupCommitFile,
	}, db, exportSvc, companyStore)

	// Bootstrap: seed superadmin if no companies exist
	bootstrap(cfg, companyStore, userStore, syncStore)

	// Handlers
	h := &handler.Handlers{
		Auth:    handler.NewAuthHandler(authSvc),
		QA:      handler.NewQAHandler(qaSvc),
		Theme:   handler.NewThemeHandler(themeSvc),
		Pricing: handler.NewPricingHandler(pricingSvc),
		Article: handler.NewArticleHandler(articleSvc),
		Comment: handler.NewCommentHandler(commentSvc),
		Link:    handler.NewLinkHandler(linkSvc),
		Search:  handler.NewSearchHandler(searchSvc),
		Export:  handler.NewExportHandler(exportSvc, userStore),
		Sync:    handler.NewSyncHandler(syncSvc),
		Admin:   handler.NewAdminHandler(adminSvc),
		User:    handler.NewUserHandler(userSvc),
		Call:    handler.NewCallHandler(callSvc),
		Backup:  handler.NewBackupHandler(snapshotSvc),
	}

	var syncRepo domain.SyncRepository = syncStore
	// The backup API key is read per-request so it can be rotated/revoked
	// without restarting the process.
	backupKey := func() string { return os.Getenv("BACKUP_API_KEY") }
	router := handler.NewRouter(h, jwtMgr, syncRepo, backupKey)

	log.Println("KnowledgeOS API starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func bootstrap(cfg *config.Config, companies *store.CompanyStore, users *store.UserStore, syncRepo *store.SyncStore) {
	ctx := context.Background()
	count, err := companies.Count(ctx)
	if err != nil || count > 0 {
		return
	}

	if cfg.SuperadminEmail == "" || cfg.SuperadminPassword == "" {
		log.Println("No companies found and SUPERADMIN_EMAIL/PASSWORD not set, skipping bootstrap")
		return
	}

	log.Println("First run detected, bootstrapping superadmin...")

	hash, err := auth.HashPassword(cfg.SuperadminPassword)
	if err != nil {
		log.Fatalf("Failed to hash superadmin password: %v", err)
	}

	company := &domain.Company{
		Name: "Default",
		Tier: "local",
	}
	if err := companies.Create(ctx, company); err != nil {
		log.Fatalf("Failed to create default company: %v", err)
	}

	if err := syncRepo.InitSequence(ctx, company.ID); err != nil {
		log.Fatalf("Failed to init sync sequence: %v", err)
	}

	user := &domain.User{
		Email:        cfg.SuperadminEmail,
		PasswordHash: hash,
		Role:         domain.RoleSuperadmin,
		CompanyID:    &company.ID,
		IsActive:     true,
	}
	if err := users.Create(ctx, user); err != nil {
		log.Fatalf("Failed to create superadmin user: %v", err)
	}

	log.Printf("Superadmin created: %s", cfg.SuperadminEmail)
}
