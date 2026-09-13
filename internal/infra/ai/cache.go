package ai

import (
	"strings"
	"sync"
	"time"

	"study-gin-clerk/internal/types"
)

type cacheItem struct {
	models    []types.AIModel
	expiresAt time.Time
}

// MemoryCache provides thread-safe in-memory caching for model lists partitioned by (userID, provider).
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string]cacheItem
	ttl    time.Duration
	stopCh chan struct{}
}

// NewMemoryCache creates a new MemoryCache with the specified TTL.
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	c := &MemoryCache{
		items:  make(map[string]cacheItem),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	// Periodic cleanup of expired items every 10 minutes
	go c.startCleanup(10 * time.Minute)

	return c
}

// NewMemoryCacheDefault creates a MemoryCache with default 15-minute TTL and a cleanup function (for DI).
func NewMemoryCacheDefault() (*MemoryCache, func()) {
	c := NewMemoryCache(15 * time.Minute)
	return c, func() {
		c.Close()
	}
}

// Close stops the periodic background cleanup goroutine.
func (c *MemoryCache) Close() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

// Get retrieves cached models for a specific user and provider.
func (c *MemoryCache) Get(userID string, provider types.Provider) ([]types.AIModel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := userID + ":" + string(provider)
	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}

	result := make([]types.AIModel, len(item.models))
	copy(result, item.models)
	return result, true
}

// Set stores models for a specific user and provider with TTL.
func (c *MemoryCache) Set(userID string, provider types.Provider, models []types.AIModel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := userID + ":" + string(provider)
	c.items[key] = cacheItem{
		models:    models,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Purge removes cached models for a specific user and provider.
func (c *MemoryCache) Purge(userID string, provider types.Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, userID+":"+string(provider))
}

// PurgeUser removes all cached models for all providers of a specific user.
func (c *MemoryCache) PurgeUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := userID + ":"
	for k := range c.items {
		if strings.HasPrefix(k, prefix) {
			delete(c.items, k)
		}
	}
}

func (c *MemoryCache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case now := <-ticker.C:
			c.mu.Lock()
			for k, v := range c.items {
				if now.After(v.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
