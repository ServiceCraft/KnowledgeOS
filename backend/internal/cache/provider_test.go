package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/cache"
	"github.com/knowledgeos/backend/internal/testutil"
)

func TestMemoryProviderCompanyExists(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	checker := &testutil.FakeCompanyChecker{ExistsMap: map[uuid.UUID]bool{id: true}}
	provider := cache.NewMemoryProvider(checker, time.Minute, time.Second)

	tests := []struct {
		name string // что проверяем
		run  func(t *testing.T) error
	}{
		{
			name: "cache miss loads checker",
			run: func(t *testing.T) error {
				ok, err := provider.CompanyExists(ctx, id)
				if err != nil || !ok {
					t.Fatalf("exists = %v err = %v", ok, err)
				}
				return nil
			},
		},
		{
			name: "invalidate forces reload path",
			run: func(t *testing.T) error {
				if err := provider.InvalidateCompany(ctx, id); err != nil {
					return err
				}
				ok, err := provider.CompanyExists(ctx, id)
				if err != nil || !ok {
					t.Fatalf("exists after invalidate = %v err = %v", ok, err)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(t); err != nil {
				t.Fatal(err)
			}
		})
	}
}
