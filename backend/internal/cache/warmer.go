package cache

import (
	"context"
	"log"
	"sync"
	"time"
)

type WarmupJob struct {
	Key      string
	Data     interface{}
	TTL      time.Duration
	Priority int
}

type WarmupStrategy struct {
	Jobs            []WarmupJob
	BatchSize       int
	ConcurrentJobs  int
	WarmupInterval  time.Duration
	HealthCheckFunc func() bool
	UseWorkerPool   bool
	UseScheduler    bool
}

type CacheWarmer struct {
	cache    Cache
	strategy *WarmupStrategy
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}

	workerPool    *WorkerPool
	scheduler     *JobScheduler
	priorityQueue *PriorityQueue
}

func NewCacheWarmer(cache Cache, strategy *WarmupStrategy) *CacheWarmer {
	if strategy == nil {
		strategy = &WarmupStrategy{
			BatchSize:      10,
			ConcurrentJobs: 3,
			WarmupInterval: 5 * time.Minute,
			UseWorkerPool:  true,
			UseScheduler:   true,
		}
	}

	cw := &CacheWarmer{
		cache:    cache,
		strategy: strategy,
		stopCh:   make(chan struct{}),
	}

	if strategy.UseWorkerPool {
		cw.workerPool = NewWorkerPool(strategy.ConcurrentJobs, cache)
	}

	if strategy.UseScheduler {
		cw.scheduler = NewJobScheduler(cw, cw.workerPool)
		if strategy.WarmupInterval > 0 {
			cw.scheduler.AddIntervalTrigger("default", strategy.WarmupInterval, cw.shouldWarmup)
		}
	}

	cw.priorityQueue = NewPriorityQueue()

	for _, job := range strategy.Jobs {
		cw.priorityQueue.Push(job)
	}

	return cw
}

func (cw *CacheWarmer) AddWarmupJob(job WarmupJob) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.priorityQueue != nil {
		cw.priorityQueue.Push(job)
	} else {
		cw.strategy.Jobs = append(cw.strategy.Jobs, job)
		for i := len(cw.strategy.Jobs) - 1; i > 0; i-- {
			if cw.strategy.Jobs[i].Priority > cw.strategy.Jobs[i-1].Priority {
				cw.strategy.Jobs[i], cw.strategy.Jobs[i-1] = cw.strategy.Jobs[i-1], cw.strategy.Jobs[i]
			} else {
				break
			}
		}
	}

	if cw.scheduler != nil && cw.scheduler.IsRunning() {
		cw.scheduler.ScheduleJob(job)
	}

	log.Printf("📝 Added warmup job: %s (priority: %d)", job.Key, job.Priority)
}

func (cw *CacheWarmer) Start(ctx context.Context) {
	cw.mu.Lock()
	if cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = true
	cw.mu.Unlock()

	jobCount := len(cw.strategy.Jobs)
	if cw.priorityQueue != nil {
		jobCount = cw.priorityQueue.Len()
	}
	log.Printf("🔥 Starting cache warmer with %d jobs", jobCount)

	if cw.workerPool != nil {
		cw.workerPool.Start()
	}

	if cw.scheduler != nil {
		cw.scheduler.Start()
	}

	if cw.strategy.UseScheduler && cw.scheduler != nil {
		jobs := cw.priorityQueue.GetJobs()
		cw.scheduler.ScheduleJobs(jobs)
		return
	}

	go cw.warmCache(ctx)

	if cw.strategy.WarmupInterval > 0 {
		ticker := time.NewTicker(cw.strategy.WarmupInterval)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if cw.shouldWarmup() {
						go cw.warmCache(ctx)
					}
				case <-cw.stopCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.running {
		return
	}

	cw.running = false

	if cw.scheduler != nil {
		cw.scheduler.Stop()
	}

	if cw.workerPool != nil {
		cw.workerPool.Stop()
	}

	close(cw.stopCh)
	log.Printf("🛑 Cache warmer stopped")
}

func (cw *CacheWarmer) WarmCacheManually(ctx context.Context) {
	if cw.strategy.UseScheduler && cw.scheduler != nil {
		processed := cw.scheduler.ProcessScheduledJobs()
		log.Printf("🔥 Manual cache warming processed %d jobs", processed)
		return
	}

	go cw.warmCache(ctx)
}

func (cw *CacheWarmer) warmCache(ctx context.Context) {
	cw.mu.RLock()
	var jobs []WarmupJob

	if cw.priorityQueue != nil {
		jobs = cw.priorityQueue.GetJobs()
	} else {
		jobs = make([]WarmupJob, len(cw.strategy.Jobs))
		copy(jobs, cw.strategy.Jobs)
	}

	batchSize := cw.strategy.BatchSize
	concurrentJobs := cw.strategy.ConcurrentJobs
	useWorkerPool := cw.strategy.UseWorkerPool && cw.workerPool != nil
	cw.mu.RUnlock()

	if len(jobs) == 0 {
		return
	}

	log.Printf("🔥 Warming cache with %d jobs (batch: %d, concurrent: %d, worker pool: %v)",
		len(jobs), batchSize, concurrentJobs, useWorkerPool)

	if useWorkerPool {
		submitted := cw.workerPool.SubmitJobs(jobs)
		log.Printf("✅ Submitted %d jobs to worker pool", submitted)
		return
	}

	for i := 0; i < len(jobs); i += batchSize {
		end := i + batchSize
		if end > len(jobs) {
			end = len(jobs)
		}

		batch := jobs[i:end]
		cw.processBatch(ctx, batch, concurrentJobs)

		select {
		case <-ctx.Done():
			log.Printf("Cache warming cancelled")
			return
		default:
		}
	}

	log.Printf("✅ Cache warming completed")
}

func (cw *CacheWarmer) processBatch(ctx context.Context, jobs []WarmupJob, concurrency int) {
	jobCh := make(chan WarmupJob, len(jobs))
	var wg sync.WaitGroup

	for i := 0; i < concurrency && i < len(jobs); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				select {
				case <-ctx.Done():
					return
				default:
					cw.processJob(job)
				}
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	wg.Wait()
}

func (cw *CacheWarmer) processJob(job WarmupJob) {
	err := cw.cache.Set(job.Key, job.Data, job.TTL)
	if err != nil {
		log.Printf("Failed to warm cache key %s: %v", job.Key, err)
	}
}

func (cw *CacheWarmer) shouldWarmup() bool {
	if cw.strategy.HealthCheckFunc != nil {
		return cw.strategy.HealthCheckFunc()
	}

	if healthChecker, ok := cw.cache.(interface{ Health() error }); ok {
		return healthChecker.Health() == nil
	}

	return true
}

func (cw *CacheWarmer) GetStats() map[string]interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	totalJobs := len(cw.strategy.Jobs)
	if cw.priorityQueue != nil {
		totalJobs += cw.priorityQueue.Len()
	}

	stats := map[string]interface{}{
		"running":         cw.running,
		"interval":        cw.strategy.WarmupInterval.String(),
		"total_jobs":      totalJobs,
		"batch_size":      cw.strategy.BatchSize,
		"concurrent_jobs": cw.strategy.ConcurrentJobs,
		"use_worker_pool": cw.strategy.UseWorkerPool,
		"use_scheduler":   cw.strategy.UseScheduler,
	}

	if cw.workerPool != nil {
		workerStats := cw.workerPool.GetStats()
		stats["worker_pool"] = workerStats
	}

	if cw.scheduler != nil {
		schedulerStats := cw.scheduler.GetStats()
		stats["scheduler"] = schedulerStats
	}

	if cw.priorityQueue != nil {
		stats["priority_queue_size"] = cw.priorityQueue.Len()
	}

	return stats
}
