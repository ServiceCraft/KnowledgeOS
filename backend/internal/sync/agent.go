package sync

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/service"
)

type Agent struct {
	cfg       *config.Config
	syncSvc   *service.SyncService
	companyID uuid.UUID
	pusher    *Pusher
	puller    *Puller
}

func NewAgent(cfg *config.Config, syncSvc *service.SyncService, companyID uuid.UUID) *Agent {
	return &Agent{
		cfg:       cfg,
		syncSvc:   syncSvc,
		companyID: companyID,
		pusher:    NewPusher(cfg.CloudAPIURL, cfg.CloudAPIKey, syncSvc),
		puller:    NewPuller(cfg.CloudAPIURL, cfg.CloudAPIKey, syncSvc),
	}
}

func (a *Agent) Run() {
	interval := time.Duration(a.cfg.SyncIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Sync agent started, interval: %s, company: %s", interval, a.companyID)

	// Run once immediately
	a.cycle()

	for range ticker.C {
		a.cycle()
	}
}

func (a *Agent) cycle() {
	log.Println("Sync cycle starting...")

	if err := a.pusher.Push(a.companyID); err != nil {
		log.Printf("Push error: %v", err)
	}

	if err := a.puller.Pull(a.companyID); err != nil {
		log.Printf("Pull error: %v", err)
	}

	log.Println("Sync cycle complete")
}
