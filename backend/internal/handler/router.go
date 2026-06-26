package handler

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
)

type Handlers struct {
	Auth     *AuthHandler
	QA       *QAHandler
	Theme    *ThemeHandler
	Pricing  *PricingHandler
	Article  *ArticleHandler
	Comment  *CommentHandler
	Link     *LinkHandler
	Search   *SearchHandler
	Export   *ExportHandler
	Sync     *SyncHandler
	Admin    *AdminHandler
	User     *UserHandler
	Call     *CallHandler
	Backup   *BackupHandler
	Bot      *BotHandler
	RAG      *RAGHandler
	Chat     *ChatHandler
	Channels *ChannelWebhookHandler
}

// writeRoles are allowed to mutate business entities. Viewer (Оператор) is
// intentionally excluded — it is a read-only role.
var writeRoles = []domain.Role{domain.RoleEditor, domain.RoleAdmin, domain.RoleSuperadmin}

// NewRouter executes the handler.NewRouter operation.
func NewRouter(h *Handlers, jwtMgr *auth.JWTManager, syncRepo domain.SyncRepository, backupKey func() string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Post("/auth/login", h.Auth.Login)
		r.Post("/auth/refresh", h.Auth.Refresh)
		r.Post("/webhooks/{channel}/{company_id}", h.Channels.Handle)

		// Protected (JWT + tenant). All reads available to any authenticated
		// role (viewer included); writes are gated to editor and above.
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(jwtMgr))
			r.Use(middleware.Tenant)

			r.Post("/auth/logout", h.Auth.Logout)

			// --- Reads (viewer+) ---
			r.Get("/qa", h.QA.List)
			r.Get("/qa/{id}", h.QA.Get)
			r.Get("/qa/{id}/mentions", h.Call.ListMentionsForQA)
			r.Get("/calls/{id}", h.Call.GetCall)
			r.Get("/themes", h.Theme.List)
			r.Get("/themes/{id}", h.Theme.Get)
			r.Get("/themes/{id}/qa", h.Theme.ListQA)
			r.Get("/pricing", h.Pricing.List)
			r.Get("/pricing/{id}", h.Pricing.Get)
			r.Get("/articles", h.Article.List)
			r.Get("/articles/{id}", h.Article.Get)
			r.Get("/search", h.Search.Search)
			r.Get("/sync/status", h.Sync.Status)

			// --- Writes (editor+) ---
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(writeRoles...))

				// QA
				r.Post("/qa", h.QA.Create)
				r.Patch("/qa/{id}", h.QA.Update)
				r.Post("/qa/{id}/review", h.QA.Review)
				r.Delete("/qa/{id}", h.QA.Delete)

				// Themes
				r.Post("/themes", h.Theme.Create)
				r.Patch("/themes/{id}", h.Theme.Update)
				r.Delete("/themes/{id}", h.Theme.Delete)

				// Pricing
				r.Post("/pricing", h.Pricing.Create)
				r.Patch("/pricing/{id}", h.Pricing.Update)
				r.Delete("/pricing/{id}", h.Pricing.Delete)
				r.Post("/pricing/{id}/move", h.Pricing.Move)

				// Articles
				r.Post("/articles", h.Article.Create)
				r.Patch("/articles/{id}", h.Article.Update)
				r.Delete("/articles/{id}", h.Article.Delete)
			})

			// Comments (polymorphic): read for everyone, write for editor+.
			r.Route("/{entityType}/{entityID}/comments", func(r chi.Router) {
				r.Use(middleware.EntityType)
				r.Get("/", h.Comment.List)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(writeRoles...))
					r.Post("/", h.Comment.Create)
					r.Patch("/{id}", h.Comment.Update)
					r.Delete("/{id}", h.Comment.Delete)
				})
			})

			// Links (polymorphic): read for everyone, write for editor+.
			r.Route("/{entityType}/{entityID}/links", func(r chi.Router) {
				r.Use(middleware.EntityType)
				r.Get("/", h.Link.List)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(writeRoles...))
					r.Post("/", h.Link.Create)
					r.Delete("/{id}", h.Link.Delete)
				})
			})

			// User administration (admin + superadmin). Assigning the
			// superadmin role is further restricted inside the service.
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleSuperadmin))
				r.Get("/admin/users", h.User.List)
				r.Post("/admin/users", h.User.Create)
				r.Patch("/admin/users/{id}", h.User.Update)
				r.Delete("/admin/users/{id}", h.User.Delete)
			})

			// Bot settings and tenant secrets — admin + superadmin.
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleSuperadmin))
				r.Get("/admin/bot/settings", h.Bot.GetSettings)
				r.Put("/admin/bot/settings", h.Bot.UpdateSettings)
				r.Get("/admin/bot/secrets", h.Bot.ListSecrets)
				r.Put("/admin/bot/secrets/{kind}", h.Bot.SetSecret)
				r.Delete("/admin/bot/secrets/{kind}", h.Bot.DeleteSecret)
				r.Get("/admin/bot/channels/status", h.Channels.Status)
				r.Post("/admin/bot/rag/reindex", h.RAG.Reindex)
				r.Get("/admin/bot/rag/index-status", h.RAG.Status)
				r.Post("/admin/bot/rag/search", h.RAG.Search)
			})

			// Bot playground chat — authenticated users within the tenant.
			r.Route("/admin/bot/chat", func(r chi.Router) {
				r.Post("/sessions", h.Chat.CreateSession)
				r.Get("/sessions", h.Chat.ListSessions)
				r.Get("/sessions/{id}", h.Chat.GetSession)
				r.Post("/sessions/{id}/messages", h.Chat.SendMessage)
				r.Post("/sessions/{id}/messages/stream", h.Chat.StreamMessage)
			})

			// Import/Export of the knowledge base — superadmin only.
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin))
				r.Get("/export", h.Export.Export)
				r.Post("/import", h.Export.Import)
			})

			// Company administration — superadmin only.
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin))
				r.Get("/admin/companies", h.Admin.ListCompanies)
				r.Post("/admin/companies", h.Admin.CreateCompany)
				r.Get("/admin/companies/{id}", h.Admin.GetCompany)
				r.Patch("/admin/companies/{id}", h.Admin.UpdateCompany)
				r.Delete("/admin/companies/{id}", h.Admin.DeleteCompany)
				r.Post("/admin/companies/{id}/admin", h.Admin.CreateCompanyAdmin)
			})
		})

		// Sync routes (API Key auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.SyncAPIKeyAuth(syncRepo))
			r.Post("/sync/push", h.Sync.Push)
			r.Get("/sync/pull", h.Sync.Pull)
		})

		// Backup snapshot endpoint (dedicated backup API key, not JWT).
		r.Group(func(r chi.Router) {
			r.Use(middleware.BackupAPIKeyAuth(backupKey))
			r.Get("/backup/snapshot", h.Backup.Snapshot)
		})
	})

	return r
}
