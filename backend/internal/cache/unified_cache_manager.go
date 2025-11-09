package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type UnifiedCacheManager struct {
	integratedWarmer *IntegratedCacheWarmer
	legacyWarmer     *CacheWarmer
	useIntegrated    bool
	ctx              context.Context
}

type CacheWarmingMode int

const (
	ModeAuto        CacheWarmingMode = iota 
	ModeLegacy                              
	ModeIntegrated                          
	ModeLocalOnly                           
	ModeDistributed                         
)

type CacheWarmingConfig struct {
	Mode          CacheWarmingMode
	RedisClient   *redis.Client
	Strategy      *WarmupStrategy
	EnableMetrics bool

	PreferDistributed    bool
	LocalFallback        bool
	DistributedThreshold int
	MaxRetries           int
}

func NewUnifiedCacheManager(cache Cache, config *CacheWarmingConfig) *UnifiedCacheManager {
	if config == nil {
		config = &CacheWarmingConfig{
			Mode: ModeAuto,
			Strategy: &WarmupStrategy{
				BatchSize:      10,
				ConcurrentJobs: 3,
				WarmupInterval: 5 * time.Minute,
				UseWorkerPool:  true,
				UseScheduler:   true,
			},
			EnableMetrics: true,
			LocalFallback: true,
			MaxRetries:    3,
		}
	}

	ucm := &UnifiedCacheManager{
		ctx: context.Background(),
	}

	actualMode := ucm.determineMode(config, cache)
	ucm.useIntegrated = actualMode != ModeLegacy

	if ucm.useIntegrated {
		ucm.integratedWarmer = NewIntegratedCacheWarmer(cache, config.RedisClient, config.Strategy)
		log.Printf("Unified cache manager initialized in integrated mode")
	} else {
		ucm.legacyWarmer = NewCacheWarmer(cache, config.Strategy)
		log.Printf("Unified cache manager initialized in legacy mode")
	}

	return ucm
}

func (ucm *UnifiedCacheManager) determineMode(config *CacheWarmingConfig, cache Cache) CacheWarmingMode {
	switch config.Mode {
	case ModeLegacy:
		return ModeLegacy
	case ModeIntegrated, ModeLocalOnly, ModeDistributed:
		if config.RedisClient != nil {
			return config.Mode
		}
		log.Println("Redis client not available, falling back to legacy mode")
		return ModeLegacy
	case ModeAuto:
		if config.RedisClient != nil {
			
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := config.RedisClient.Ping(ctx).Err(); err == nil {
				log.Println("Auto-mode: Redis available, using integrated mode")
				return ModeIntegrated
			} else {
				log.Printf("Auto-mode: Redis test failed, using legacy mode: %v", err)
			}
		}
		return ModeLegacy
	default:
		return ModeLegacy
	}
}

func (ucm *UnifiedCacheManager) Start() error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.Start()
	}
	ucm.legacyWarmer.Start(ucm.ctx)
	return nil
}

func (ucm *UnifiedCacheManager) Stop() error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.Stop()
	}
	ucm.legacyWarmer.Stop()
	return nil
}

func (ucm *UnifiedCacheManager) WarmupCache() error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.WarmupCache()
	}
	ucm.legacyWarmer.WarmCacheManually(ucm.ctx)
	return nil
}

func (ucm *UnifiedCacheManager) EnqueueWarmupJob(key string, data interface{}, ttl time.Duration, priority int) error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.EnqueueWarmupJob(key, data, ttl, priority)
	}

	job := WarmupJob{
		Key:      key,
		Data:     data,
		TTL:      ttl,
		Priority: priority,
	}
	ucm.legacyWarmer.AddWarmupJob(job)
	return nil
}

func (ucm *UnifiedCacheManager) EnqueueBatchWarmupJob(keys []string, data interface{}, priority int) error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.EnqueueBatchWarmupJob(keys, data, priority)
	}

	for _, key := range keys {
		job := WarmupJob{
			Key:      key,
			Data:     data,
			TTL:      1 * time.Hour,
			Priority: priority,
		}
		ucm.legacyWarmer.AddWarmupJob(job)
	}
	return nil
}

func (ucm *UnifiedCacheManager) EnqueueScheduledWarmup(key string, data interface{}, ttl time.Duration, processAt time.Time, priority int) error {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.EnqueueScheduledWarmup(key, data, ttl, processAt, priority)
	}

	log.Printf("Scheduled warmup not supported in legacy mode, executing immediately for key: %s", key)
	job := WarmupJob{
		Key:      key,
		Data:     data,
		TTL:      ttl,
		Priority: priority,
	}
	ucm.legacyWarmer.AddWarmupJob(job)
	return nil
}

func (ucm *UnifiedCacheManager) EnqueueEvictionJob(key string, priority int) error {
	if !ucm.useIntegrated {
		return fmt.Errorf("eviction jobs not supported in legacy mode")
	}

	job := DistributedCacheJob{
		ID:   fmt.Sprintf("evict-%d", time.Now().UnixNano()),
		Type: CacheJobEviction,
		Payload: map[string]interface{}{
			"key": key,
		},
		MaxTries:  3,
		CreatedAt: time.Now(),
		ProcessAt: time.Now(),
		Priority:  priority,
	}

	return ucm.integratedWarmer.routeJob(job)
}

func (ucm *UnifiedCacheManager) EnqueueValidationJob(key string, expectedData interface{}, priority int) error {
	if !ucm.useIntegrated {
		return fmt.Errorf("validation jobs not supported in legacy mode")
	}

	job := DistributedCacheJob{
		ID:   fmt.Sprintf("validate-%d", time.Now().UnixNano()),
		Type: CacheJobValidation,
		Payload: map[string]interface{}{
			"key":  key,
			"data": expectedData,
		},
		MaxTries:  3,
		CreatedAt: time.Now(),
		ProcessAt: time.Now(),
		Priority:  priority,
	}

	return ucm.integratedWarmer.routeJob(job)
}

func (ucm *UnifiedCacheManager) GetMetrics() map[string]interface{} {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.GetMetrics()
	}

	stats := ucm.legacyWarmer.GetStats()
	stats["system_type"] = "legacy_memory"
	return stats
}

func (ucm *UnifiedCacheManager) GetQueueSizes() (map[string]int64, error) {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.GetQueueSizes()
	}

	return map[string]int64{"local": 0}, nil
}

func (ucm *UnifiedCacheManager) GetSystemInfo() map[string]interface{} {
	info := map[string]interface{}{
		"integrated_mode": ucm.useIntegrated,
		"running":         ucm.IsRunning(),
	}

	if ucm.useIntegrated {
		info["system_type"] = "integrated"
		info["capabilities"] = []string{
			"distributed_processing",
			"job_persistence",
			"retry_logic",
			"scheduled_jobs",
			"dead_letter_queues",
			"local_fallback",
		}

		if metrics := ucm.integratedWarmer.GetMetrics(); metrics != nil {
			info["redis_available"] = metrics["redis_available"]
			info["distributed_workers"] = metrics["distributed_workers"]
		}
	} else {
		info["system_type"] = "legacy"
		info["capabilities"] = []string{
			"local_processing",
			"worker_pool",
			"job_scheduler",
		}

		if stats := ucm.legacyWarmer.GetStats(); stats != nil {
			info["concurrent_jobs"] = stats["concurrent_jobs"]
			info["use_worker_pool"] = stats["use_worker_pool"]
		}
	}

	return info
}

func (ucm *UnifiedCacheManager) IsRunning() bool {
	if ucm.useIntegrated {
		return ucm.integratedWarmer.IsRunning()
	}
	return ucm.legacyWarmer != nil 
}

func (ucm *UnifiedCacheManager) IsIntegrated() bool {
	return ucm.useIntegrated
}

func (ucm *UnifiedCacheManager) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	if ucm.useIntegrated {
		
		if ucm.integratedWarmer.redisClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := ucm.integratedWarmer.redisClient.Ping(ctx).Err(); err != nil {
				health["redis_status"] = "unhealthy"
				health["redis_error"] = err.Error()
				health["status"] = "degraded"
			} else {
				health["redis_status"] = "healthy"
			}
		}

		if sizes, err := ucm.GetQueueSizes(); err == nil {
			health["queue_sizes"] = sizes
		}

		metrics := ucm.integratedWarmer.GetMetrics()
		health["workers_active"] = metrics["distributed_workers"]
		health["total_processed"] = metrics["processed_jobs"]
		health["total_failed"] = metrics["failed_jobs"]
	} else {
		
		health["mode"] = "legacy"
		if stats := ucm.legacyWarmer.GetStats(); stats != nil {
			health["worker_pool_running"] = stats["use_worker_pool"]
		}
	}

	return health
}

func (ucm *UnifiedCacheManager) UpdateConfiguration(newStrategy *WarmupStrategy) error {
	if newStrategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	wasRunning := ucm.IsRunning()

	if wasRunning {
		if err := ucm.Stop(); err != nil {
			return fmt.Errorf("failed to stop system for reconfiguration: %w", err)
		}
	}

	if ucm.useIntegrated {
		ucm.integratedWarmer.strategy = newStrategy
	} else {
		ucm.legacyWarmer.strategy = newStrategy
	}

	if wasRunning {
		if err := ucm.Start(); err != nil {
			return fmt.Errorf("failed to restart system after reconfiguration: %w", err)
		}
	}

	log.Println("Cache warming configuration updated successfully")
	return nil
}

func (ucm *UnifiedCacheManager) WarmupUserData(userID string, userData interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s", userID)
	return ucm.EnqueueWarmupJob(key, userData, ttl, 2) 
}

func (ucm *UnifiedCacheManager) WarmupBatchUserData(userIDs []string, userDataMap map[string]interface{}, ttl time.Duration) error {
	for _, userID := range userIDs {
		if userData, exists := userDataMap[userID]; exists {
			if err := ucm.WarmupUserData(userID, userData, ttl); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ucm *UnifiedCacheManager) WarmupPopularContent(contentType string, items []interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("popular:%s", contentType)
	return ucm.EnqueueWarmupJob(key, items, ttl, 1) 
}

func (ucm *UnifiedCacheManager) SchedulePeriodicWarmup(key string, data interface{}, ttl time.Duration, interval time.Duration) error {
	if !ucm.useIntegrated {
		return fmt.Errorf("periodic warmup requires integrated mode")
	}

	nextRun := time.Now().Add(interval)
	return ucm.EnqueueScheduledWarmup(key, data, ttl, nextRun, 3) 
}
