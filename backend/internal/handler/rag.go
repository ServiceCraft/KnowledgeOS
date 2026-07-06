package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type RAGHandler struct {
	indexer   *service.RAGIndexerService
	retriever *service.RetrieverService
}

// NewRAGHandler executes the handler.NewRAGHandler operation.
func NewRAGHandler(indexer *service.RAGIndexerService, retriever *service.RetrieverService) *RAGHandler {
	return &RAGHandler{indexer: indexer, retriever: retriever}
}

// Reindex executes the handler.RAGHandler.Reindex operation.
func (h *RAGHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.RAGHandler.Reindex")
	companyID := middleware.GetCompanyID(r.Context())
	if err := h.indexer.ReindexCompany(r.Context(), companyID); err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

// Status executes the handler.RAGHandler.Status operation.
func (h *RAGHandler) Status(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.RAGHandler.Status")
	companyID := middleware.GetCompanyID(r.Context())
	status, err := h.indexer.IndexStatus(r.Context(), companyID)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, status)
}

type ragSearchRequest struct {
	Query      string   `json:"query"`
	Types      []string `json:"types,omitempty"`
	ThemeID    *string  `json:"theme_id,omitempty"`
	VectorTopK int      `json:"vector_top_k,omitempty"`
	HybridTopK int      `json:"hybrid_top_k,omitempty"`
	Rewrite    *bool    `json:"rewrite,omitempty"`
}

// Search executes the handler.RAGHandler.Search operation.
func (h *RAGHandler) Search(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.RAGHandler.Search")
	var req ragSearchRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	retrieveReq := domain.RetrieveRequest{
		Query:      req.Query,
		VectorTopK: req.VectorTopK,
		HybridTopK: req.HybridTopK,
		Rewrite:    true,
	}
	if req.Rewrite != nil {
		retrieveReq.Rewrite = *req.Rewrite
	}
	for _, t := range req.Types {
		t = strings.TrimSpace(t)
		if t != "" {
			retrieveReq.Types = append(retrieveReq.Types, domain.KBEntityType(t))
		}
	}
	if req.ThemeID != nil && strings.TrimSpace(*req.ThemeID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*req.ThemeID))
		if err != nil {
			Error(w, http.StatusBadRequest, "invalid theme_id")
			return
		}
		retrieveReq.ThemeID = &id
	}

	companyID := middleware.GetCompanyID(r.Context())
	result, err := h.retriever.Search(r.Context(), companyID, retrieveReq)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, result)
}
