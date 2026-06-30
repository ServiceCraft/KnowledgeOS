package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/cache"
	"github.com/knowledgeos/backend/internal/channels"
	channelmax "github.com/knowledgeos/backend/internal/channels/max"
	"github.com/knowledgeos/backend/internal/channels/telegram"
	"github.com/knowledgeos/backend/internal/channels/vk"
	"github.com/knowledgeos/backend/internal/chat/tools"
	"github.com/knowledgeos/backend/internal/config"
	secretcrypto "github.com/knowledgeos/backend/internal/crypto"
	"github.com/knowledgeos/backend/internal/database"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/handler"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type App struct {
	cfg          *config.Config
	db           *gorm.DB
	ragIndexer   *service.RAGIndexerService
	workerCancel context.CancelFunc
	server       *http.Server
}

// New wires dependencies and returns a runnable application.
func New(cfg *config.Config) (*App, error) {
	applog.Configure(cfg.LogLevel, cfg.LogFormat, "knowledgeos-api")
	if cfg.BotChatDebugLog {
		log.Warn().Msg("BOT_CHAT_DEBUG_LOG enabled: chat prompts and LLM responses are written to logs (may contain PII)")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.RunMigrations(db, "migrations"); err != nil {
		return nil, err
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL())
	s := store.New(db, cfg.AppProfile)

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
	botSettingsStore := store.NewBotSettingsStore(s)
	tenantSecretStore := store.NewTenantSecretStore(s)
	kbEmbeddingStore := store.NewKBEmbeddingStore(s)
	kbIndexJobStore := store.NewKBIndexJobStore(s)
	chatStore := store.NewChatStore(s)

	companyCache := cache.NewMemoryProvider(
		companyStore,
		cfg.CompanyCachePositiveTTL(),
		cfg.CompanyCacheNegativeTTL(),
	)

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
			WorkerEnabled:      cfg.RAGWorkerEnabled,
			WorkerBatchSize:    cfg.RAGWorkerBatchSize,
			WorkerPollInterval: time.Duration(cfg.RAGWorkerPollIntervalSeconds) * time.Second,
			IndexMaxAttempts:   cfg.RAGIndexMaxAttempts,
		},
	)
	authSvc := service.NewAuthService(
		userStore,
		companyStore,
		syncStore,
		jwtMgr,
		cfg.RefreshTokenTTL(),
		cfg.AuthListCompaniesLimit,
	)
	qaSvc := service.NewQAService(qaStore, themeStore, ragIndexerSvc)
	themeSvc := service.NewThemeService(themeStore, qaStore)
	pricingSvc := service.NewPricingService(pricingStore, ragIndexerSvc)
	articleSvc := service.NewArticleService(articleStore, ragIndexerSvc)
	commentSvc := service.NewCommentService(commentStore, qaStore, articleStore, pricingStore)
	linkSvc := service.NewLinkService(linkStore, qaStore, articleStore, pricingStore)
	searchSvc := service.NewSearchService(searchStore)
	retrieverSvc := service.NewRetrieverService(kbEmbeddingStore, searchStore, llmFactory, cfg.RAGVectorTopK, cfg.RAGHybridTopK)
	chatTools := tools.NewRegistry(
		tools.NewSearchKnowledgeTool(retrieverSvc, cfg.RAGHybridTopK),
		tools.NewGetPricingTool(pricingSvc),
		tools.NewGetServiceInfoTool(articleSvc, qaSvc, pricingSvc),
		tools.NewRequestHandoffTool(),
	)
	chatSvc := service.NewChatService(chatStore, botSettingsSvc, retrieverSvc, llmFactory, chatTools, cfg.BotChatDebugLog)
	channelGateway := channels.NewGateway(
		chatStore,
		chatSvc,
		botSettingsSvc,
		tenantSecretSvc,
		telegram.New(nil),
		channelmax.New(nil),
		vk.New(nil),
	)
	handoffSvc := service.NewHandoffService(chatStore, botSettingsSvc, channelGateway, channelGateway)
	chatSvc.SetHandoffEscalator(handoffSvc)
	channelGateway.SetHandoff(handoffSvc)
	exportSvc := service.NewExportService(db, themeStore, qaStore, pricingStore, articleStore, commentStore, linkStore, callStore, mentionStore)
	syncSvc := service.NewSyncService(syncStore, themeStore, qaStore, pricingStore, articleStore, commentStore, linkStore)
	callSvc := service.NewCallService(callStore, mentionStore, qaStore)
	adminSvc := service.NewAdminService(companyStore, userStore, syncStore, companyCache)
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

	bootstrapSuperadmin(cfg, companyStore, userStore, syncStore)

	h := &handler.Handlers{
		Auth:     handler.NewAuthHandler(authSvc),
		QA:       handler.NewQAHandler(qaSvc),
		Theme:    handler.NewThemeHandler(themeSvc),
		Pricing:  handler.NewPricingHandler(pricingSvc),
		Article:  handler.NewArticleHandler(articleSvc),
		Comment:  handler.NewCommentHandler(commentSvc),
		Link:     handler.NewLinkHandler(linkSvc),
		Search:   handler.NewSearchHandler(searchSvc),
		Export:   handler.NewExportHandler(exportSvc, userStore),
		Sync:     handler.NewSyncHandler(syncSvc),
		Admin:    handler.NewAdminHandler(adminSvc),
		User:     handler.NewUserHandler(userSvc),
		Call:     handler.NewCallHandler(callSvc),
		Backup:   handler.NewBackupHandler(snapshotSvc),
		Bot:      handler.NewBotHandlerWithChannels(botSettingsSvc, tenantSecretSvc, channelGateway),
		RAG:      handler.NewRAGHandler(ragIndexerSvc, retrieverSvc),
		Chat:     handler.NewChatHandler(chatSvc),
		Handoff:  handler.NewHandoffHandler(handoffSvc),
		Channels: handler.NewChannelWebhookHandler(channelGateway),
	}

	var syncRepo domain.SyncRepository = syncStore
	backupKey := func() string { return os.Getenv("BACKUP_API_KEY") }
	dbPing := func(r *http.Request) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(r.Context())
	}
	router := handler.NewRouter(handler.RouterDeps{
		Handlers:     h,
		JWT:          jwtMgr,
		SyncRepo:     syncRepo,
		Membership:   userStore,
		CompanyCache: companyCache,
		BackupKey:    backupKey,
		DBPing:       dbPing,
	})

	return &App{
		cfg:        cfg,
		db:         db,
		ragIndexer: ragIndexerSvc,
		server: &http.Server{
			Addr:              ":8080",
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}, nil
}

// Run starts background workers and the HTTP server with graceful shutdown.
func (a *App) Run() error {
	workerCtx, cancel := context.WithCancel(context.Background())
	a.workerCancel = cancel
	if a.cfg.RAGWorkerEnabled {
		a.ragIndexer.StartWorker(workerCtx)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", a.server.Addr).Msg("knowledgeos api starting")
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		cancel()
		return err
	case <-stop:
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Info().Msg("knowledgeos api stopped")
	return nil
}

func bootstrapSuperadmin(cfg *config.Config, companies *store.CompanyStore, users *store.UserStore, syncRepo *store.SyncStore) {
	ctx := context.Background()
	count, err := companies.Count(ctx)
	if err != nil || count > 0 {
		return
	}
	if cfg.SuperadminEmail == "" || cfg.SuperadminPassword == "" {
		log.Warn().Msg("no companies found and bootstrap credentials are not set")
		return
	}

	log.Info().Msg("first run detected, bootstrapping superadmin")
	hash, err := auth.HashPassword(cfg.SuperadminPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to hash superadmin password")
	}

	company := &domain.Company{Name: "Default", Tier: "local"}
	if err := companies.Create(ctx, company); err != nil {
		log.Fatal().Err(err).Msg("failed to create default company")
	}
	if err := syncRepo.InitSequence(ctx, company.ID); err != nil {
		log.Fatal().Err(err).Msg("failed to init sync sequence")
	}

	user := &domain.User{
		Email:        cfg.SuperadminEmail,
		PasswordHash: hash,
		Role:         domain.RoleSuperadmin,
		IsActive:     true,
	}
	if err := users.Create(ctx, user); err != nil {
		log.Fatal().Err(err).Msg("failed to create superadmin user")
	}
	log.Info().Str("company_id", company.ID.String()).Msg("superadmin created")
}
