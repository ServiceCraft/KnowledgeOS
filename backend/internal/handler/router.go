package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/cache"
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
	Handoff  *HandoffHandler
	Channels *ChannelWebhookHandler
}

type RouterDeps struct {
	Handlers     *Handlers
	JWT          *auth.JWTManager
	SyncRepo     domain.SyncRepository
	Membership   domain.UserMembershipReader
	CompanyCache cache.Provider
	BackupKey    func() string
	DBPing       func(r *http.Request) error
}

// writeRoles are allowed to mutate business entities. Viewer (Оператор) is
// intentionally excluded — it is a read-only role.
var writeRoles = []domain.Role{domain.RoleEditor, domain.RoleAdmin, domain.RoleSuperadmin}

// NewRouter executes the handler.NewRouter operation.
func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(chiMiddleware.Heartbeat("/healthz"))

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if deps.DBPing != nil {
			if err := deps.DBPing(r); err != nil {
				Error(w, http.StatusServiceUnavailable, "database unavailable")
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	h := deps.Handlers

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(20, time.Minute))
			r.Post("/auth/login", h.Auth.Login)
			r.Post("/auth/refresh", h.Auth.Refresh)
		})
		r.Post("/webhooks/{channel}/{company_id}", h.Channels.Handle)

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(deps.JWT))
			r.Post("/auth/logout", h.Auth.Logout)
			r.Get("/auth/companies", h.Auth.ListCompanies)

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

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(deps.JWT))
			r.Use(middleware.Tenant(deps.Membership, deps.CompanyCache))

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

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(writeRoles...))

				r.Post("/qa", h.QA.Create)
				r.Patch("/qa/{id}", h.QA.Update)
				r.Post("/qa/{id}/review", h.QA.Review)
				r.Delete("/qa/{id}", h.QA.Delete)

				r.Post("/themes", h.Theme.Create)
				r.Patch("/themes/{id}", h.Theme.Update)
				r.Delete("/themes/{id}", h.Theme.Delete)

				r.Post("/pricing", h.Pricing.Create)
				r.Patch("/pricing/{id}", h.Pricing.Update)
				r.Delete("/pricing/{id}", h.Pricing.Delete)
				r.Post("/pricing/{id}/move", h.Pricing.Move)

				r.Post("/articles", h.Article.Create)
				r.Patch("/articles/{id}", h.Article.Update)
				r.Delete("/articles/{id}", h.Article.Delete)
			})

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

			r.Route("/{entityType}/{entityID}/links", func(r chi.Router) {
				r.Use(middleware.EntityType)
				r.Get("/", h.Link.List)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(writeRoles...))
					r.Post("/", h.Link.Create)
					r.Delete("/{id}", h.Link.Delete)
				})
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleSuperadmin))
				r.Get("/admin/users", h.User.List)
				r.Post("/admin/users", h.User.Create)
				r.Patch("/admin/users/{id}", h.User.Update)
				r.Delete("/admin/users/{id}", h.User.Delete)
			})

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

			r.Route("/admin/bot/chat", func(r chi.Router) {
				r.Post("/sessions", h.Chat.CreateSession)
				r.Get("/sessions", h.Chat.ListSessions)
				r.Get("/sessions/{id}", h.Chat.GetSession)
				r.Post("/sessions/{id}/messages", h.Chat.SendMessage)
				r.Post("/sessions/{id}/messages/stream", h.Chat.StreamMessage)
			})

			r.Route("/admin/bot/handoff", func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleViewer, domain.RoleAdmin, domain.RoleSuperadmin))
				r.Get("/sessions", h.Handoff.ListSessions)
				r.Get("/sessions/metrics", h.Handoff.Metrics)
				r.Get("/sessions/{id}", h.Handoff.GetSession)
				r.Post("/sessions/{id}/claim", h.Handoff.Claim)
				r.Post("/sessions/{id}/messages", h.Handoff.SendMessage)
				r.Post("/sessions/{id}/release", h.Handoff.Release)
				r.Post("/sessions/{id}/close", h.Handoff.Close)
				r.Post("/sessions/{id}/escalate", h.Handoff.Escalate)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleSuperadmin))
				r.Get("/export", h.Export.Export)
				r.Post("/import", h.Export.Import)
			})

		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.SyncAPIKeyAuth(deps.SyncRepo))
			r.Post("/sync/push", h.Sync.Push)
			r.Get("/sync/pull", h.Sync.Pull)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.BackupAPIKeyAuth(deps.BackupKey))
			r.Get("/backup/snapshot", h.Backup.Snapshot)
		})
	})

	return r
}
