package cache

import (
	"valancis-backend/pkg/cache"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type memoryCache struct {
	store *gocache.Cache
}

// NewMemoryCache creates a new in-memory cache service
// defaultExpiration: default TTL for items
// cleanupInterval: how often to scan for expired items
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) cache.CacheService {
	return &memoryCache{
		store: gocache.New(defaultExpiration, cleanupInterval),
	}
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

func (c *memoryCache) Set(key string, value interface{}, duration time.Duration) {
	c.store.Set(key, value, duration)
}

func (c *memoryCache) Delete(key string) {
	c.store.Delete(key)
}

func (c *memoryCache) Flush() {
	c.store.Flush()
}
