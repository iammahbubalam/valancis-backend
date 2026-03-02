package cache

import "time"

// CacheService defines the behavior for caching mechanisms
type CacheService interface {
	// Get retrieves a value from the cache
	// Returns value, true if found
	// Returns nil, false if not found
	Get(key string) (interface{}, bool)

	// Set adds a value to the cache with a duration
	Set(key string, value interface{}, duration time.Duration)

	// Delete removes a value from the cache
	Delete(key string)

	// Flush removes all items
	Flush()
}
