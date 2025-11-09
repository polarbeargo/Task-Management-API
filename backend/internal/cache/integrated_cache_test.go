package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestIntegratedCacheWarmer_Basic(t *testing.T) {
	
	cache := NewMultiLevelCache(nil) 

	config := &CacheWarmingConfig{
		Mode: ModeAuto,
		Strategy: &WarmupStrategy{
			BatchSize:      5,
			ConcurrentJobs: 2,
			WarmupInterval: 1 * time.Minute,
			UseWorkerPool:  true,
			UseScheduler:   false, 
		},
	}

	manager := NewUnifiedCacheManager(cache, config)

	if manager.IsIntegrated() {
		t.Error("Expected legacy mode with nil Redis client")
	}

	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	if !manager.IsRunning() {
		t.Error("Manager should be running after start")
	}

	if err := manager.EnqueueWarmupJob("test:key", "test_data", 1*time.Hour, 1); err != nil {
		t.Errorf("Failed to enqueue warmup job: %v", err)
	}

	keys := []string{"batch:1", "batch:2", "batch:3"}
	if err := manager.EnqueueBatchWarmupJob(keys, "batch_data", 2); err != nil {
		t.Errorf("Failed to enqueue batch job: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	systemInfo := manager.GetSystemInfo()
	if systemInfo["system_type"] != "legacy" {
		t.Errorf("Expected legacy system type, got %v", systemInfo["system_type"])
	}

	health := manager.HealthCheck()
	if health["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", health["status"])
	}

	if err := manager.Stop(); err != nil {
		t.Errorf("Failed to stop manager: %v", err)
	}
}

func TestIntegratedCacheWarmer_WithRedis(t *testing.T) {
	
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, 
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integrated test")
	}

	defer func() {
		client.FlushDB(context.Background())
		client.Close()
	}()

	redisConfig := &CacheConfig{
		Addr: "localhost:6379",
		DB:   1, 
	}
	redisCache := NewRedisCache(redisConfig)
	cache := NewMultiLevelCache(redisCache)

	config := &CacheWarmingConfig{
		Mode:        ModeIntegrated,
		RedisClient: client,
		Strategy: &WarmupStrategy{
			BatchSize:      3,
			ConcurrentJobs: 2,
			WarmupInterval: 1 * time.Minute,
			UseWorkerPool:  true,
			UseScheduler:   false,
		},
		EnableMetrics: true,
		LocalFallback: true,
	}

	manager := NewUnifiedCacheManager(cache, config)

	if !manager.IsIntegrated() {
		t.Error("Expected integrated mode with Redis client")
	}

	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start integrated manager: %v", err)
	}
	defer manager.Stop()

	if err := manager.EnqueueWarmupJob("redis:test:key", "redis_test_data", 30*time.Minute, 1); err != nil {
		t.Errorf("Failed to enqueue Redis warmup job: %v", err)
	}

	futureTime := time.Now().Add(2 * time.Second)
	if err := manager.EnqueueScheduledWarmup("redis:scheduled", "scheduled_data", 1*time.Hour, futureTime, 2); err != nil {
		t.Errorf("Failed to enqueue scheduled job: %v", err)
	}

	keys := []string{"redis:batch:1", "redis:batch:2", "redis:batch:3"}
	if err := manager.EnqueueBatchWarmupJob(keys, "redis_batch_data", 2); err != nil {
		t.Errorf("Failed to enqueue Redis batch job: %v", err)
	}

	if err := manager.EnqueueEvictionJob("redis:old:key", 3); err != nil {
		t.Errorf("Failed to enqueue eviction job: %v", err)
	}

	if err := manager.EnqueueValidationJob("redis:test:key", "redis_test_data", 3); err != nil {
		t.Errorf("Failed to enqueue validation job: %v", err)
	}

	time.Sleep(3 * time.Second)

	metrics := manager.GetMetrics()
	if metrics["system_type"] != "integrated_redis_local" {
		t.Errorf("Expected integrated system type, got %v", metrics["system_type"])
	}

	queueSizes, err := manager.GetQueueSizes()
	if err != nil {
		t.Errorf("Failed to get queue sizes: %v", err)
	}

	t.Logf("Queue sizes: %+v", queueSizes)
	t.Logf("Metrics: %+v", metrics)

	var retrievedData interface{}
	if err := cache.Get("redis:test:key", &retrievedData); err != nil {
		t.Logf("Note: Key not found in cache (might be expected): %v", err)
	} else {
		t.Logf("Successfully retrieved cached data: %v", retrievedData)
	}

	health := manager.HealthCheck()
	if health["redis_status"] != "healthy" {
		t.Errorf("Expected healthy Redis status, got %v", health["redis_status"])
	}

	if err := manager.WarmupUserData("test_user", map[string]string{"name": "Test User"}, 1*time.Hour); err != nil {
		t.Errorf("Failed to warmup user data: %v", err)
	}

	popularItems := []interface{}{"item1", "item2", "item3"}
	if err := manager.WarmupPopularContent("test_products", popularItems, 30*time.Minute); err != nil {
		t.Errorf("Failed to warmup popular content: %v", err)
	}

	time.Sleep(1 * time.Second)

	finalMetrics := manager.GetMetrics()
	t.Logf("Final metrics: %+v", finalMetrics)
}

func TestUnifiedCacheManager_ModeDetection(t *testing.T) {
	cache := NewMultiLevelCache(nil)

	tests := []struct {
		name               string
		mode               CacheWarmingMode
		redisClient        *redis.Client
		expectedMode       CacheWarmingMode
		shouldBeIntegrated bool
	}{
		{
			name:               "Force Legacy Mode",
			mode:               ModeLegacy,
			redisClient:        &redis.Client{}, 
			expectedMode:       ModeLegacy,
			shouldBeIntegrated: false,
		},
		{
			name:               "Auto Mode No Redis",
			mode:               ModeAuto,
			redisClient:        nil,
			expectedMode:       ModeLegacy,
			shouldBeIntegrated: false,
		},
		{
			name:               "Integrated Mode No Redis",
			mode:               ModeIntegrated,
			redisClient:        nil,
			expectedMode:       ModeLegacy,
			shouldBeIntegrated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CacheWarmingConfig{
				Mode:        tt.mode,
				RedisClient: tt.redisClient,
				Strategy: &WarmupStrategy{
					BatchSize:      5,
					ConcurrentJobs: 2,
				},
			}

			manager := NewUnifiedCacheManager(cache, config)

			if manager.IsIntegrated() != tt.shouldBeIntegrated {
				t.Errorf("Expected integrated: %v, got: %v", tt.shouldBeIntegrated, manager.IsIntegrated())
			}

			systemInfo := manager.GetSystemInfo()
			t.Logf("System info for %s: %+v", tt.name, systemInfo)
		})
	}
}

func TestUnifiedCacheManager_ConvenienceMethods(t *testing.T) {
	cache := NewMultiLevelCache(nil)
	config := &CacheWarmingConfig{Mode: ModeLegacy}
	manager := NewUnifiedCacheManager(cache, config)

	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()

	if err := manager.WarmupUserData("user123", map[string]string{"name": "John"}, 1*time.Hour); err != nil {
		t.Errorf("Failed to warmup user data: %v", err)
	}

	userIDs := []string{"user1", "user2", "user3"}
	userData := map[string]interface{}{
		"user1": map[string]string{"name": "Alice"},
		"user2": map[string]string{"name": "Bob"},
		"user3": map[string]string{"name": "Charlie"},
	}

	if err := manager.WarmupBatchUserData(userIDs, userData, 2*time.Hour); err != nil {
		t.Errorf("Failed to warmup batch user data: %v", err)
	}

	popularItems := []interface{}{"product1", "product2", "product3"}
	if err := manager.WarmupPopularContent("electronics", popularItems, 30*time.Minute); err != nil {
		t.Errorf("Failed to warmup popular content: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	metrics := manager.GetMetrics()
	t.Logf("Convenience methods metrics: %+v", metrics)
}

func BenchmarkUnifiedCacheManager_EnqueueJob(b *testing.B) {
	cache := NewMultiLevelCache(nil)
	config := &CacheWarmingConfig{Mode: ModeLegacy}
	manager := NewUnifiedCacheManager(cache, config)

	if err := manager.Start(); err != nil {
		b.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench:key:%d", i)
			manager.EnqueueWarmupJob(key, "bench_data", 1*time.Hour, 2)
			i++
		}
	})
}

func BenchmarkUnifiedCacheManager_BatchJob(b *testing.B) {
	cache := NewMultiLevelCache(nil)
	config := &CacheWarmingConfig{Mode: ModeLegacy}
	manager := NewUnifiedCacheManager(cache, config)

	if err := manager.Start(); err != nil {
		b.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()

	keys := make([]string, 10)
	for i := range keys {
		keys[i] = fmt.Sprintf("batch:key:%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.EnqueueBatchWarmupJob(keys, "batch_data", 2)
	}
}
