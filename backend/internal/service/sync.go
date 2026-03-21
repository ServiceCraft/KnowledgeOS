package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type SyncService struct {
	syncRepo domain.SyncRepository
	themes   domain.ThemeRepository
	qa       domain.QAPairRepository
	pricing  domain.PricingNodeRepository
	articles domain.ArticleRepository
	comments domain.CommentRepository
	links    domain.EntityLinkRepository
}

func NewSyncService(syncRepo domain.SyncRepository, themes domain.ThemeRepository, qa domain.QAPairRepository, pricing domain.PricingNodeRepository, articles domain.ArticleRepository, comments domain.CommentRepository, links domain.EntityLinkRepository) *SyncService {
	return &SyncService{
		syncRepo: syncRepo,
		themes:   themes,
		qa:       qa,
		pricing:  pricing,
		articles: articles,
		comments: comments,
		links:    links,
	}
}

type SyncPushPayload struct {
	CompanyID    uuid.UUID            `json:"company_id"`
	Themes       []domain.Theme       `json:"themes"`
	QAPairs      []domain.QAPair      `json:"qa_pairs"`
	PricingNodes []domain.PricingNode `json:"pricing_nodes"`
	Articles     []domain.Article     `json:"articles"`
	Comments     []domain.Comment     `json:"comments"`
	Links        []domain.EntityLink  `json:"links"`
	MaxSeq       int64                `json:"max_seq"`
}

type SyncPullRequest struct {
	SinceSeq int64 `json:"since_seq"`
}

func (s *SyncService) Status(ctx context.Context, companyID uuid.UUID) (*domain.SyncStatus, error) {
	return s.syncRepo.GetStatus(ctx, companyID)
}

func (s *SyncService) GatherPush(ctx context.Context, companyID uuid.UUID) (*SyncPushPayload, error) {
	wm, err := s.syncRepo.GetWatermark(ctx, companyID)
	if err != nil {
		return nil, err
	}

	sinceVer := wm.LastLocalSeq

	themes, _ := s.themes.ListSince(ctx, companyID, sinceVer)
	qas, _ := s.qa.ListSince(ctx, companyID, sinceVer)
	nodes, _ := s.pricing.ListSince(ctx, companyID, sinceVer)
	articles, _ := s.articles.ListSince(ctx, companyID, sinceVer)
	comments, _ := s.comments.ListSince(ctx, companyID, sinceVer)
	links, _ := s.links.ListSince(ctx, companyID, sinceVer)

	var maxSeq int64
	for _, t := range themes {
		if t.SyncVersion > maxSeq {
			maxSeq = t.SyncVersion
		}
	}
	for _, q := range qas {
		if q.SyncVersion > maxSeq {
			maxSeq = q.SyncVersion
		}
	}
	for _, n := range nodes {
		if n.SyncVersion > maxSeq {
			maxSeq = n.SyncVersion
		}
	}
	for _, a := range articles {
		if a.SyncVersion > maxSeq {
			maxSeq = a.SyncVersion
		}
	}
	for _, c := range comments {
		if c.SyncVersion > maxSeq {
			maxSeq = c.SyncVersion
		}
	}
	for _, l := range links {
		if l.SyncVersion > maxSeq {
			maxSeq = l.SyncVersion
		}
	}

	return &SyncPushPayload{
		CompanyID:    companyID,
		Themes:       themes,
		QAPairs:      qas,
		PricingNodes: nodes,
		Articles:     articles,
		Comments:     comments,
		Links:        links,
		MaxSeq:       maxSeq,
	}, nil
}

func (s *SyncService) ApplyPull(ctx context.Context, companyID uuid.UUID, payload *SyncPushPayload) error {
	for i := range payload.Themes {
		_ = s.themes.ApplyRemote(ctx, companyID, &payload.Themes[i])
	}
	for i := range payload.QAPairs {
		_ = s.qa.ApplyRemote(ctx, companyID, &payload.QAPairs[i])
	}
	for i := range payload.PricingNodes {
		_ = s.pricing.ApplyRemote(ctx, companyID, &payload.PricingNodes[i])
	}
	for i := range payload.Articles {
		_ = s.articles.ApplyRemote(ctx, companyID, &payload.Articles[i])
	}
	for i := range payload.Comments {
		_ = s.comments.ApplyRemote(ctx, companyID, &payload.Comments[i])
	}
	for i := range payload.Links {
		_ = s.links.ApplyRemote(ctx, companyID, &payload.Links[i])
	}

	if payload.MaxSeq > 0 {
		now := time.Now()
		wm, _ := s.syncRepo.GetWatermark(ctx, companyID)
		wm.LastCloudSeq = payload.MaxSeq
		wm.LastSyncAt = &now
		_ = s.syncRepo.UpdateWatermark(ctx, wm)
	}

	return nil
}

func (s *SyncService) CompletePush(ctx context.Context, companyID uuid.UUID, maxSeq int64) error {
	now := time.Now()
	wm, _ := s.syncRepo.GetWatermark(ctx, companyID)
	wm.LastLocalSeq = maxSeq
	wm.LastSyncAt = &now
	return s.syncRepo.UpdateWatermark(ctx, wm)
}
