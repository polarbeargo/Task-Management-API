package cache

import (
	"sync"
	"time"
)

type MemoryCache struct {
	store sync.Map
	mutex sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{}

	go cache.cleanup()

	return cache
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	expiration := time.Now().Add(ttl)
	c.store.Store(key, &cacheItem{
		value:      value,
		expiration: expiration,
	})
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	item, exists := c.store.Load(key)
	if !exists {
		return nil, false
	}

	cacheItem := item.(*cacheItem)

	if time.Now().After(cacheItem.expiration) {
		c.store.Delete(key)
		return nil, false
	}

	return cacheItem.value, true
}

func (c *MemoryCache) Delete(key string) {
	c.store.Delete(key)
}

func (c *MemoryCache) DeletePattern(pattern string) {
	c.store.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if matchPattern(keyStr, pattern) {
			c.store.Delete(key)
		}
		return true
	})
}

func (c *MemoryCache) Clear() {
	c.store = sync.Map{}
}

func (c *MemoryCache) Stats() map[string]interface{} {
	count := 0
	c.store.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"items": count,
		"type":  "memory",
	}
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.store.Range(func(key, value interface{}) bool {
			cacheItem := value.(*cacheItem)
			if now.After(cacheItem.expiration) {
				c.store.Delete(key)
			}
			return true
		})
	}
}

func matchPattern(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(text) >= len(prefix) && text[:len(prefix)] == prefix
	}

	return text == pattern
}
