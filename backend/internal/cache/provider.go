package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"golang.org/x/sync/singleflight"
)

// Provider resolves company existence with TTL caching.
type Provider interface {
	CompanyExists(ctx context.Context, id uuid.UUID) (bool, error)
	InvalidateCompany(ctx context.Context, id uuid.UUID) error
}

type cacheEntry struct {
	exists    bool
	expiresAt time.Time
}

// MemoryProvider is an in-process TTL cache backed by CompanyExistenceChecker.
type MemoryProvider struct {
	checker     domain.CompanyExistenceChecker
	positiveTTL time.Duration
	negativeTTL time.Duration
	mu          sync.RWMutex
	entries     map[uuid.UUID]cacheEntry
	group       singleflight.Group
}

// NewMemoryProvider executes the cache.NewMemoryProvider operation.
func NewMemoryProvider(checker domain.CompanyExistenceChecker, positiveTTL, negativeTTL time.Duration) *MemoryProvider {
	if positiveTTL <= 0 {
		positiveTTL = 5 * time.Minute
	}
	if negativeTTL <= 0 {
		negativeTTL = 30 * time.Second
	}
	return &MemoryProvider{
		checker:     checker,
		positiveTTL: positiveTTL,
		negativeTTL: negativeTTL,
		entries:     make(map[uuid.UUID]cacheEntry),
	}
}

// CompanyExists executes the cache.MemoryProvider.CompanyExists operation.
func (p *MemoryProvider) CompanyExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}

	now := time.Now()
	p.mu.RLock()
	if entry, ok := p.entries[id]; ok && now.Before(entry.expiresAt) {
		p.mu.RUnlock()
		return entry.exists, nil
	}
	p.mu.RUnlock()

	key := id.String()
	v, err, _ := p.group.Do(key, func() (interface{}, error) {
		exists, err := p.checker.Exists(ctx, id)
		if err != nil {
			return false, err
		}
		ttl := p.positiveTTL
		if !exists {
			ttl = p.negativeTTL
		}
		p.mu.Lock()
		p.entries[id] = cacheEntry{exists: exists, expiresAt: time.Now().Add(ttl)}
		p.mu.Unlock()
		return exists, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

// InvalidateCompany executes the cache.MemoryProvider.InvalidateCompany operation.
func (p *MemoryProvider) InvalidateCompany(_ context.Context, id uuid.UUID) error {
	p.mu.Lock()
	delete(p.entries, id)
	p.mu.Unlock()
	return nil
}
