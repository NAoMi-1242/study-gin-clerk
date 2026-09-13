package ai

import (
	"strings"
	"sync"
	"time"

	"study-gin-clerk/internal/model"
)

type cacheItem struct {
	models    []model.AIModelInfo
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
func (c *MemoryCache) Get(userID, provider string) ([]model.AIModelInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := userID + ":" + provider
	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}

	result := make([]model.AIModelInfo, len(item.models))
	copy(result, item.models)
	return result, true
}

// Set stores models for a specific user and provider with TTL.
func (c *MemoryCache) Set(userID, provider string, models []model.AIModelInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := userID + ":" + provider
	c.items[key] = cacheItem{
		models:    models,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Purge removes cached models for a specific user and provider (or all providers for the user if provider is empty).
func (c *MemoryCache) Purge(userID, provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if provider == "" {
		prefix := userID + ":"
		for k := range c.items {
			if strings.HasPrefix(k, prefix) {
				delete(c.items, k)
			}
		}
		return
	}

	delete(c.items, userID+":"+provider)
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

