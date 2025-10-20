package cache

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	if config.Addr != "localhost:6379" {
		t.Errorf("Expected Addr to be localhost:6379, got %s", config.Addr)
	}

	if config.Password != "" {
		t.Errorf("Expected Password to be empty, got %s", config.Password)
	}

	if config.DB != 0 {
		t.Errorf("Expected DB to be 0, got %d", config.DB)
	}

	if config.PoolSize != 10 {
		t.Errorf("Expected PoolSize to be 10, got %d", config.PoolSize)
	}

	if config.MinIdleConns != 5 {
		t.Errorf("Expected MinIdleConns to be 5, got %d", config.MinIdleConns)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.DialTimeout != 5*time.Second {
		t.Errorf("Expected DialTimeout to be 5s, got %v", config.DialTimeout)
	}

	if config.ReadTimeout != 3*time.Second {
		t.Errorf("Expected ReadTimeout to be 3s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 3*time.Second {
		t.Errorf("Expected WriteTimeout to be 3s, got %v", config.WriteTimeout)
	}
}

func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)

	config := &CacheConfig{
		Addr:         mr.Addr(),
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	cache := NewRedisCache(config)
	return cache, mr
}

func TestNewRedisCache_WithNilConfig(t *testing.T) {
	cache := NewRedisCache(nil)

	if cache == nil {
		t.Error("Expected cache to be created with default config")
	}

	if cache.client == nil {
		t.Error("Expected Redis client to be initialized")
	}
}

func TestNewRedisCache_WithCustomConfig(t *testing.T) {
	config := &CacheConfig{
		Addr:         "localhost:6379",
		Password:     "test-password",
		DB:           1,
		PoolSize:     20,
		MinIdleConns: 10,
		MaxRetries:   5,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	cache := NewRedisCache(config)

	if cache == nil {
		t.Error("Expected cache to be created")
	}

	if cache.client == nil {
		t.Error("Expected Redis client to be initialized")
	}
}

func TestRedisCache_SetAndGet(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := testData{Name: "test", Value: 42}
	key := "test:key"

	err := cache.Set(key, original, time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	var retrieved testData
	err = cache.Get(key, &retrieved)
	if err != nil {
		t.Fatalf("Failed to get from cache: %v", err)
	}

	if retrieved.Name != original.Name {
		t.Errorf("Expected Name %s, got %s", original.Name, retrieved.Name)
	}

	if retrieved.Value != original.Value {
		t.Errorf("Expected Value %d, got %d", original.Value, retrieved.Value)
	}
}

func TestRedisCache_Get_CacheMiss(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	var result string
	err := cache.Get("non-existent-key", &result)

	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestRedisCache_Set_InvalidData(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	ch := make(chan int)
	err := cache.Set("test:key", ch, time.Minute)

	if err == nil {
		t.Error("Expected error when setting unmarshalable data")
	}
}

func TestRedisCache_Get_InvalidJSON(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	mr.Set("test:invalid", "invalid-json")

	var result map[string]interface{}
	err := cache.Get("test:invalid", &result)

	if err == nil {
		t.Error("Expected error when getting invalid JSON")
	}
}

func TestRedisCache_Delete(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	key := "test:delete"
	data := "test-data"

	err := cache.Set(key, data, time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	var retrieved string
	err = cache.Get(key, &retrieved)
	if err != nil {
		t.Fatalf("Failed to get from cache: %v", err)
	}

	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete from cache: %v", err)
	}

	err = cache.Get(key, &retrieved)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestRedisCache_DeletePattern(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	keys := []string{"test:pattern:1", "test:pattern:2", "test:other:1"}
	for _, key := range keys {
		err := cache.Set(key, "data", time.Minute)
		if err != nil {
			t.Fatalf("Failed to set cache key %s: %v", key, err)
		}
	}

	err := cache.DeletePattern("test:pattern:*")
	if err != nil {
		t.Fatalf("Failed to delete pattern: %v", err)
	}

	var result string
	for _, key := range []string{"test:pattern:1", "test:pattern:2"} {
		err = cache.Get(key, &result)
		if err != ErrCacheMiss {
			t.Errorf("Expected key %s to be deleted, but got: %v", key, err)
		}
	}

	err = cache.Get("test:other:1", &result)
	if err != nil {
		t.Errorf("Expected key test:other:1 to still exist, got: %v", err)
	}
}

func TestRedisCache_Exists(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	key := "test:exists"

	exists, err := cache.Exists(key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	err = cache.Set(key, "data", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	exists, err = cache.Exists(key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}
}

func TestRedisCache_SetWithTags(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	key := "test:tagged"
	data := "tagged-data"
	tags := []string{"user:123", "post:456"}

	err := cache.SetWithTags(key, data, time.Minute, tags)
	if err != nil {
		t.Fatalf("Failed to set with tags: %v", err)
	}

	var retrieved string
	err = cache.Get(key, &retrieved)
	if err != nil {
		t.Fatalf("Failed to get tagged data: %v", err)
	}

	if retrieved != data {
		t.Errorf("Expected %s, got %s", data, retrieved)
	}

	for _, tag := range tags {
		tagKey := "tag:" + tag
		exists, err := cache.Exists(tagKey)
		if err != nil {
			t.Fatalf("Failed to check tag existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected tag %s to exist", tagKey)
		}
	}
}

func TestRedisCache_InvalidateByTag(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	tag := "user:123"
	keys := []string{"test:tagged:1", "test:tagged:2"}

	for _, key := range keys {
		err := cache.SetWithTags(key, "data", time.Minute, []string{tag})
		if err != nil {
			t.Fatalf("Failed to set tagged key %s: %v", key, err)
		}
	}

	err := cache.InvalidateByTag(tag)
	if err != nil {
		t.Fatalf("Failed to invalidate by tag: %v", err)
	}

	var result string
	for _, key := range keys {
		err = cache.Get(key, &result)
		if err != ErrCacheMiss {
			t.Errorf("Expected key %s to be invalidated, got: %v", key, err)
		}
	}
}

func TestRedisCache_Health(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	err := cache.Health()
	if err != nil {
		t.Errorf("Expected healthy cache, got error: %v", err)
	}

	mr.Close()

	err = cache.Health()
	if err == nil {
		t.Error("Expected unhealthy cache after closing Redis")
	}
}

func TestRedisCache_Stats(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	stats := cache.Stats()

	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if len(stats) == 0 {
		t.Log("Stats is empty, which is expected with miniredis mock")
	}
}

func TestRedisCache_Close(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()

	err := cache.Close()
	if err != nil {
		t.Errorf("Failed to close cache: %v", err)
	}

	err = cache.Set("test", "data", time.Minute)
	if err == nil {
		t.Error("Expected error when using cache after close")
	}
}

func BenchmarkRedisCache_Set(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()

	config := &CacheConfig{Addr: mr.Addr()}
	cache := NewRedisCache(config)

	data := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cache.Set("benchmark:key", data, time.Minute)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}
}

func BenchmarkRedisCache_Get(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()

	config := &CacheConfig{Addr: mr.Addr()}
	cache := NewRedisCache(config)

	data := map[string]string{"key": "value"}
	err := cache.Set("benchmark:key", data, time.Minute)
	if err != nil {
		b.Fatalf("Failed to set cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]string
		err := cache.Get("benchmark:key", &result)
		if err != nil {
			b.Fatalf("Failed to get cache: %v", err)
		}
	}
}

func TestErrCacheMiss(t *testing.T) {
	if ErrCacheMiss.Error() != "cache miss" {
		t.Errorf("Expected ErrCacheMiss message to be 'cache miss', got '%s'", ErrCacheMiss.Error())
	}
}

func TestErrCacheDown(t *testing.T) {
	if ErrCacheDown.Error() != "cache unavailable" {
		t.Errorf("Expected ErrCacheDown message to be 'cache unavailable', got '%s'", ErrCacheDown.Error())
	}
}
