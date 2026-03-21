package handler

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
)

type Handlers struct {
	Auth    *AuthHandler
	QA      *QAHandler
	Theme   *ThemeHandler
	Pricing *PricingHandler
	Article *ArticleHandler
	Comment *CommentHandler
	Link    *LinkHandler
	Search  *SearchHandler
	Export  *ExportHandler
	Sync    *SyncHandler
	Admin   *AdminHandler
}

func NewRouter(h *Handlers, jwtMgr *auth.JWTManager, syncRepo domain.SyncRepository) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Post("/auth/login", h.Auth.Login)
		r.Post("/auth/refresh", h.Auth.Refresh)

		// Protected (JWT + tenant)
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(jwtMgr))
			r.Use(middleware.Tenant)

			r.Post("/auth/logout", h.Auth.Logout)

			// QA
			r.Get("/qa", h.QA.List)
			r.Post("/qa", h.QA.Create)
			r.Get("/qa/{id}", h.QA.Get)
			r.Patch("/qa/{id}", h.QA.Update)
			r.Delete("/qa/{id}", h.QA.Delete)

			// Themes
			r.Get("/themes", h.Theme.List)
			r.Post("/themes", h.Theme.Create)
			r.Get("/themes/{id}", h.Theme.Get)
			r.Patch("/themes/{id}", h.Theme.Update)
			r.Delete("/themes/{id}", h.Theme.Delete)
			r.Get("/themes/{id}/qa", h.Theme.ListQA)

			// Pricing
			r.Get("/pricing", h.Pricing.List)
			r.Post("/pricing", h.Pricing.Create)
			r.Get("/pricing/{id}", h.Pricing.Get)
			r.Patch("/pricing/{id}", h.Pricing.Update)
			r.Delete("/pricing/{id}", h.Pricing.Delete)
			r.Post("/pricing/{id}/move", h.Pricing.Move)

			// Articles
			r.Get("/articles", h.Article.List)
			r.Post("/articles", h.Article.Create)
			r.Get("/articles/{id}", h.Article.Get)
			r.Patch("/articles/{id}", h.Article.Update)
			r.Delete("/articles/{id}", h.Article.Delete)

			// Comments (polymorphic)
			r.Route("/{entityType}/{entityID}/comments", func(r chi.Router) {
				r.Use(middleware.EntityType)
				r.Get("/", h.Comment.List)
				r.Post("/", h.Comment.Create)
				r.Patch("/{id}", h.Comment.Update)
				r.Delete("/{id}", h.Comment.Delete)
			})

			// Links (polymorphic)
			r.Route("/{entityType}/{entityID}/links", func(r chi.Router) {
				r.Use(middleware.EntityType)
				r.Get("/", h.Link.List)
				r.Post("/", h.Link.Create)
				r.Delete("/{id}", h.Link.Delete)
			})

			// Search
			r.Get("/search", h.Search.Search)

			// Export/Import (admin+)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleSuperadmin))
				r.Get("/export", h.Export.Export)
				r.Post("/import", h.Export.Import)
			})

			// Sync status (JWT-authenticated)
			r.Get("/sync/status", h.Sync.Status)

			// Superadmin routes
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
	})

	return r
}
