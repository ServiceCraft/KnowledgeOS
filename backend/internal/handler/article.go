package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type ArticleHandler struct {
	svc *service.ArticleService
}

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	filter := domain.ArticleFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
		Query: r.URL.Query().Get("query"),
	}

	items, total, err := h.svc.List(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "article not found")
		return
	}
	JSON(w, http.StatusOK, item)
}

func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var article domain.Article
	if err := Decode(r, &article); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	article.CreatedBy = &userID
	article.UpdatedBy = &userID

	if err := h.svc.Create(r.Context(), companyID, &article); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, article)
}

func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var article domain.Article
	if err := Decode(r, &article); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	article.ID = id
	article.UpdatedBy = &userID

	if err := h.svc.Update(r.Context(), companyID, &article); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, article)
}

func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), companyID, id); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
