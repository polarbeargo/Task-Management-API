package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type Cache interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string, dest interface{}) error
	Delete(key string) error
	DeletePattern(pattern string) error
	Exists(key string) (bool, error)
	Stats() map[string]interface{}
	Health() error
	Close() error
}

type MultiLevelCache struct {
	l1 *MemoryCache 
	l2 *RedisCache  
}

func NewMultiLevelCache(redisCache *RedisCache) *MultiLevelCache {
	return &MultiLevelCache{
		l1: NewMemoryCache(),
		l2: redisCache,
	}
}

func (c *MultiLevelCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.l1.Set(key, value, ttl)

	if c.l2 != nil {
		return c.l2.Set(key, value, ttl)
	}

	return nil
}

func (c *MultiLevelCache) Get(key string, dest interface{}) error {
	if value, found := c.l1.Get(key); found {
		return copyValue(value, dest)
	}

	if c.l2 != nil {
		err := c.l2.Get(key, dest)
		if err == nil {
			c.l1.Set(key, dest, 5*time.Minute) 
		}
		return err
	}

	return ErrCacheMiss
}

func (c *MultiLevelCache) Delete(key string) error {
	c.l1.Delete(key)

	if c.l2 != nil {
		return c.l2.Delete(key)
	}

	return nil
}

func (c *MultiLevelCache) DeletePattern(pattern string) error {
	c.l1.DeletePattern(pattern)

	if c.l2 != nil {
		return c.l2.DeletePattern(pattern)
	}

	return nil
}

func (c *MultiLevelCache) Exists(key string) (bool, error) {
	if _, found := c.l1.Get(key); found {
		return true, nil
	}

	if c.l2 != nil {
		return c.l2.Exists(key)
	}

	return false, nil
}

func (c *MultiLevelCache) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"l1": c.l1.Stats(),
	}

	if c.l2 != nil {
		stats["l2"] = c.l2.Stats()
	}

	return stats
}

func (c *MultiLevelCache) Health() error {
	if c.l2 != nil {
		return c.l2.Health()
	}

	return nil
}

func (c *MultiLevelCache) Close() error {
	if c.l2 != nil {
		return c.l2.Close()
	}

	return nil
}

func copyValue(src, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer, got %T", dest)
	}

	if destValue.IsNil() {
		return fmt.Errorf("destination pointer is nil")
	}

	destElem := destValue.Elem()
	if !destElem.CanSet() {
		return fmt.Errorf("destination is not settable")
	}

	if destElem.Type() == reflect.TypeOf((*interface{})(nil)).Elem() {
		return copyValueViaJSON(src, dest)
	}

	return copyValueViaJSON(src, dest)
}

func copyValueViaJSON(src, dest interface{}) error {
	jsonData, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal source value: %w", err)
	}

	err = json.Unmarshal(jsonData, dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}

	return nil
}
