package cache

import (
	"sync"
	"time"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// Cache provides thread-safe in-memory caching
type Cache struct {
	items map[string]CacheItem
	mutex sync.RWMutex
}

// New creates a new cache instance
func New() *Cache {
	cache := &Cache{
		items: make(map[string]CacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Set stores a value in the cache with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.Expiration) {
		// Item expired, remove it
		delete(c.items, key)
		return nil, false
	}

	return item.Value, true
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]CacheItem)
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mutex.Lock()
			now := time.Now()
			for key, item := range c.items {
				if now.After(item.Expiration) {
					delete(c.items, key)
				}
			}
			c.mutex.Unlock()
		}
	}
}
