package monitoring

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	mu              sync.RWMutex
	RequestCount    int64            `json:"request_count"`
	RequestDuration time.Duration    `json:"avg_request_duration_ms"`
	ActiveRequests  int64            `json:"active_requests"`
	ErrorCount      int64            `json:"error_count"`
	StatusCodes     map[string]int64 `json:"status_codes"`
	Endpoints       map[string]int64 `json:"endpoint_calls"`
	StartTime       time.Time        `json:"start_time"`
	LastRequest     time.Time        `json:"last_request"`
	totalDuration   time.Duration
}

type HealthChecker struct {
	checks map[string]HealthCheck
	mu     sync.RWMutex
}

type HealthCheck struct {
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
	LastRun time.Time `json:"last_run"`
}

type HealthCheckFunc func(ctx context.Context) error

var globalMetrics = &Metrics{
	StatusCodes: make(map[string]int64),
	Endpoints:   make(map[string]int64),
	StartTime:   time.Now(),
}

var globalHealthChecker = &HealthChecker{
	checks: make(map[string]HealthCheck),
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		globalMetrics.mu.Lock()
		globalMetrics.ActiveRequests++
		globalMetrics.mu.Unlock()

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		endpoint := c.Request.Method + " " + c.FullPath()

		globalMetrics.mu.Lock()
		globalMetrics.RequestCount++
		globalMetrics.ActiveRequests--
		globalMetrics.totalDuration += duration
		globalMetrics.RequestDuration = globalMetrics.totalDuration / time.Duration(globalMetrics.RequestCount)
		globalMetrics.LastRequest = time.Now()

		statusStr := http.StatusText(statusCode)
		if statusCode >= 400 {
			globalMetrics.ErrorCount++
		}
		globalMetrics.StatusCodes[statusStr]++

		globalMetrics.Endpoints[endpoint]++
		globalMetrics.mu.Unlock()
	}
}

func GetMetrics() *Metrics {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()

	metrics := &Metrics{
		RequestCount:    globalMetrics.RequestCount,
		RequestDuration: globalMetrics.RequestDuration,
		ActiveRequests:  globalMetrics.ActiveRequests,
		ErrorCount:      globalMetrics.ErrorCount,
		StatusCodes:     make(map[string]int64),
		Endpoints:       make(map[string]int64),
		StartTime:       globalMetrics.StartTime,
		LastRequest:     globalMetrics.LastRequest,
	}

	for k, v := range globalMetrics.StatusCodes {
		metrics.StatusCodes[k] = v
	}
	for k, v := range globalMetrics.Endpoints {
		metrics.Endpoints[k] = v
	}

	return metrics
}

type SystemMetrics struct {
	Uptime         time.Duration `json:"uptime"`
	MemoryUsage    MemoryStats   `json:"memory"`
	GoroutineCount int           `json:"goroutine_count"`
	CPUCount       int           `json:"cpu_count"`
	GoVersion      string        `json:"go_version"`
}

type MemoryStats struct {
	Alloc        uint64 `json:"alloc_mb"`
	TotalAlloc   uint64 `json:"total_alloc_mb"`
	Sys          uint64 `json:"sys_mb"`
	NumGC        uint32 `json:"num_gc"`
	NextGC       uint64 `json:"next_gc_mb"`
	LastGC       string `json:"last_gc"`
	GCPauseTotal string `json:"gc_pause_total"`
}

func GetSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		Uptime: time.Since(globalMetrics.StartTime),
		MemoryUsage: MemoryStats{
			Alloc:        bToMb(m.Alloc),
			TotalAlloc:   bToMb(m.TotalAlloc),
			Sys:          bToMb(m.Sys),
			NumGC:        m.NumGC,
			NextGC:       bToMb(m.NextGC),
			LastGC:       time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
			GCPauseTotal: time.Duration(m.PauseTotalNs).String(),
		},
		GoroutineCount: runtime.NumGoroutine(),
		CPUCount:       runtime.NumCPU(),
		GoVersion:      runtime.Version(),
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func RegisterHealthCheck(name string, checkFunc HealthCheckFunc) {
	globalHealthChecker.mu.Lock()
	defer globalHealthChecker.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := "healthy"
	message := ""

	if err := checkFunc(ctx); err != nil {
		status = "unhealthy"
		message = err.Error()
	}

	globalHealthChecker.checks[name] = HealthCheck{
		Name:    name,
		Status:  status,
		Message: message,
		LastRun: time.Now(),
	}
}

func RunHealthChecks() map[string]HealthCheck {
	globalHealthChecker.mu.Lock()
	defer globalHealthChecker.mu.Unlock()

	results := make(map[string]HealthCheck)

	for name := range globalHealthChecker.checks {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		check := globalHealthChecker.checks[name]
		check.LastRun = time.Now()

		results[name] = check
		globalHealthChecker.checks[name] = check

		_ = ctx
	}

	return results
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := GetMetrics()
		systemMetrics := GetSystemMetrics()

		response := gin.H{
			"application": metrics,
			"system":      systemMetrics,
			"timestamp":   time.Now(),
		}

		c.JSON(http.StatusOK, response)
	}
}

func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := RunHealthChecks()

		overallStatus := "healthy"
		for _, check := range checks {
			if check.Status != "healthy" {
				overallStatus = "unhealthy"
				break
			}
		}

		response := gin.H{
			"status":    overallStatus,
			"timestamp": time.Now(),
			"checks":    checks,
			"uptime":    time.Since(globalMetrics.StartTime).String(),
		}

		status := http.StatusOK
		if overallStatus != "healthy" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, response)
	}
}

func ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := RunHealthChecks()

		ready := true
		for _, check := range checks {
			if check.Status != "healthy" {
				ready = false
				break
			}
		}

		if ready {
			c.JSON(http.StatusOK, gin.H{
				"status":    "ready",
				"timestamp": time.Now(),
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not ready",
				"timestamp": time.Now(),
			})
		}
	}
}

func LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now(),
			"uptime":    time.Since(globalMetrics.StartTime).String(),
		})
	}
}
