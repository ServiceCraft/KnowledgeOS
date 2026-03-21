package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/service"
)

type Pusher struct {
	cloudURL string
	apiKey   string
	syncSvc  *service.SyncService
}

func NewPusher(cloudURL, apiKey string, syncSvc *service.SyncService) *Pusher {
	return &Pusher{cloudURL: cloudURL, apiKey: apiKey, syncSvc: syncSvc}
}

func (p *Pusher) Push(companyID uuid.UUID) error {
	ctx := context.Background()

	payload, err := p.syncSvc.GatherPush(ctx, companyID)
	if err != nil {
		return fmt.Errorf("gather push: %w", err)
	}

	if payload.MaxSeq == 0 {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cloudURL+"/api/v1/sync/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("push request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("push failed with status %d", resp.StatusCode)
	}

	return p.syncSvc.CompletePush(ctx, companyID, payload.MaxSeq)
}
