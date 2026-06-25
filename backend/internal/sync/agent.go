package sync

import (
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/rs/zerolog/log"
)

type Agent struct {
	cfg       *config.Config
	syncSvc   *service.SyncService
	companyID uuid.UUID
	pusher    *Pusher
	puller    *Puller
}

// NewAgent executes the sync.NewAgent operation.
func NewAgent(cfg *config.Config, syncSvc *service.SyncService, companyID uuid.UUID) *Agent {
	return &Agent{
		cfg:       cfg,
		syncSvc:   syncSvc,
		companyID: companyID,
		pusher:    NewPusher(cfg.CloudAPIURL, cfg.CloudAPIKey, syncSvc),
		puller:    NewPuller(cfg.CloudAPIURL, cfg.CloudAPIKey, syncSvc),
	}
}

// Run executes the sync.Agent.Run operation.
func (a *Agent) Run() {
	interval := time.Duration(a.cfg.SyncIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", interval).
		Str("company_id", a.companyID.String()).
		Msg("sync agent started")

	// Run once immediately
	a.cycle()

	for range ticker.C {
		a.cycle()
	}
}

func (a *Agent) cycle() {
	log.Debug().Str("company_id", a.companyID.String()).Msg("sync cycle starting")

	if err := a.pusher.Push(a.companyID); err != nil {
		log.Error().Err(err).Str("company_id", a.companyID.String()).Msg("sync push failed")
	}

	if err := a.puller.Pull(a.companyID); err != nil {
		log.Error().Err(err).Str("company_id", a.companyID.String()).Msg("sync pull failed")
	}

	log.Debug().Str("company_id", a.companyID.String()).Msg("sync cycle completed")
}
