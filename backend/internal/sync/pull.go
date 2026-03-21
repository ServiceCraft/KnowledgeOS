package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/service"
)

type Puller struct {
	cloudURL string
	apiKey   string
	syncSvc  *service.SyncService
}

func NewPuller(cloudURL, apiKey string, syncSvc *service.SyncService) *Puller {
	return &Puller{cloudURL: cloudURL, apiKey: apiKey, syncSvc: syncSvc}
}

type pullResponse struct {
	Data *service.SyncPushPayload `json:"data"`
}

func (p *Puller) Pull(companyID uuid.UUID) error {
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cloudURL+"/api/v1/sync/pull", nil)
	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("pull request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pull failed with status %d", resp.StatusCode)
	}

	var result pullResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode pull response: %w", err)
	}

	if result.Data == nil {
		return nil
	}

	return p.syncSvc.ApplyPull(ctx, companyID, result.Data)
}
