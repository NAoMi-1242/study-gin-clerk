package ai

import (
	"strings"
	"sync"
	"time"

	"study-gin-clerk/internal/model"
)

type cacheItem struct {
	models    []model.AIModel
	expiresAt time.Time
}

// MemoryCache provides thread-safe in-memory caching for model lists partitioned by (userID, provider).
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

// NewMemoryCache creates a new MemoryCache with the specified TTL.
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	c := &MemoryCache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}

	// Periodic cleanup of expired items every 10 minutes
	go c.startCleanup(10 * time.Minute)

	return c
}

// NewMemoryCacheDefault creates a MemoryCache with default 15-minute TTL (for DI).
func NewMemoryCacheDefault() *MemoryCache {
	return NewMemoryCache(15 * time.Minute)
}

// Get retrieves cached models for a specific user and provider.
func (c *MemoryCache) Get(userID string, provider model.Provider) ([]model.AIModel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := userID + ":" + string(provider)
	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}

	result := make([]model.AIModel, len(item.models))
	copy(result, item.models)
	return result, true
}

// Set stores models for a specific user and provider with TTL.
func (c *MemoryCache) Set(userID string, provider model.Provider, models []model.AIModel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := userID + ":" + string(provider)
	c.items[key] = cacheItem{
		models:    models,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Purge removes cached models for a specific user and provider.
func (c *MemoryCache) Purge(userID string, provider model.Provider) {
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
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
