package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task-manager/backend/internal/cache"
	"task-manager/backend/internal/config"
	"task-manager/backend/internal/handlers"
	"task-manager/backend/internal/middleware"
	"task-manager/backend/internal/monitoring"
	"task-manager/backend/internal/repositories"
	"task-manager/backend/internal/services"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// Application holds all application dependencies and state
type Application struct {
	Config       *config.Config
	DB           *gorm.DB
	Cache        cache.Cache
	CacheManager *cache.UnifiedCacheManager
	Redis        *redis.Client
	Router       *gin.Engine
	Server       *http.Server

	// Services
	TaskService     services.TaskService
	AuthService     services.AuthService
	UserService     services.UserService
	RegisterService services.RegisterService
	AuthzService    services.AuthorizationService
}

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	app, err := initializeApplication(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize application: %v", err)
	}

	app.setupRoutes()
	app.startServer()
}

func initializeApplication(cfg *config.Config) (*Application, error) {
	app := &Application{
		Config: cfg,
	}

	log.Println("🚀 Initializing Task Manager Backend...")
	log.Printf("📋 Environment: %s", cfg.Server.Environment)

	dbCfg := repositories.NewDatabaseConfig()
	db, err := dbCfg.Connect()
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	app.DB = db

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	log.Println("✅ Database connected and configured")

	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis unavailable: %v (continuing with memory cache only)", err)
		redisClient = nil
	} else {
		app.Redis = redisClient
		log.Println("✅ Redis connected")
	}

	if redisClient != nil {
		// Multi-level cache with Redis L2
		redisCacheConfig := &cache.CacheConfig{
			Addr:         cfg.GetRedisAddr(),
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		}
		redisCache := cache.NewRedisCache(redisCacheConfig)
		app.Cache = cache.NewMultiLevelCache(redisCache)
		log.Println("✅ Multi-level cache initialized (Memory L1 + Redis L2)")
	} else {
		// Redis-only cache (will use default config with localhost)
		app.Cache = cache.NewRedisCache(cache.DefaultCacheConfig())
		log.Println("✅ Redis cache initialized (fallback mode)")
	}

	// Initialize Unified Cache Manager (auto-detects Redis availability and mode)
	if app.Cache != nil {
		warmupStrategy := &cache.WarmupStrategy{
			BatchSize:      50,
			ConcurrentJobs: 5,
			WarmupInterval: 15 * time.Minute,
			UseWorkerPool:  true,
			UseScheduler:   true,
		}

		cacheConfig := &cache.CacheWarmingConfig{
			Mode:          cache.ModeAuto, // Auto-detect Redis availability
			RedisClient:   redisClient,    // Can be nil, will fallback to legacy mode
			Strategy:      warmupStrategy,
			EnableMetrics: true,
			LocalFallback: true,
			MaxRetries:    3,
		}

		app.CacheManager = cache.NewUnifiedCacheManager(app.Cache, cacheConfig)

		if err := app.CacheManager.Start(); err != nil {
			log.Printf("⚠️  Failed to start cache manager: %v", err)
			app.CacheManager = nil
		} else {
			if app.CacheManager.IsIntegrated() {
				log.Println("✅ Unified cache manager started in INTEGRATED mode (Redis + Distributed Workers)")
			} else {
				log.Println("✅ Unified cache manager started in LEGACY mode (Local Workers)")
			}
		}
	}

	// Initialize Services
	app.AuthzService = services.NewAuthorizationService(db)
	app.AuthService = services.NewAuthService()
	app.UserService = services.NewUserService()
	app.RegisterService = services.NewRegisterService()

	// Task service with optional caching
	taskServiceImpl := services.NewTaskService()
	if multiCache, ok := app.Cache.(*cache.MultiLevelCache); ok {
		app.TaskService = services.NewCachedTaskService(taskServiceImpl, multiCache)
		log.Println("✅ Cached task service initialized")
	} else {
		app.TaskService = taskServiceImpl
		log.Println("✅ Task service initialized")
	}

	log.Println("✅ All services initialized")

	return app, nil
}

func (app *Application) setupRoutes() {
	r := gin.New()

	// Global middleware stack (order matters!)
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(monitoring.MetricsMiddleware())
	r.Use(middleware.RecoveryWithLog())
	r.Use(middleware.SecureHeader())

	// Rate limiting
	rateLimit := rate.Limit(float64(app.Config.RateLimit.RequestsPerMin) / 60.0)
	r.Use(middleware.RateLimiter(rateLimit, app.Config.RateLimit.BurstSize))

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://host.docker.internal"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health and monitoring endpoints (no auth required)
	r.GET("/health", app.healthHandler())
	r.GET("/ready", app.readinessHandler())
	r.GET("/metrics", monitoring.MetricsHandler())

	v1 := r.Group("/api/v1")

	// Public authentication routes (no auth required)
	authRoutes := v1.Group("/auth")
	{
		authHandler := handlers.NewAuthHandler(app.DB, app.AuthService)
		refreshHandler := handlers.NewRefreshHandler(app.DB, app.AuthService)
		registrationHandler := handlers.NewRegisterHandler(app.DB, app.RegisterService)

		authRoutes.POST("/register", registrationHandler.Registration)
		authRoutes.POST("/login", authHandler.Token)
		authRoutes.POST("/refresh", refreshHandler.Refresh)
	}

	// Protected routes (require authentication)
	protected := v1.Group("")
	protected.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{}))
	{
		// Task routes
		taskHandler := handlers.NewTaskHandler(app.DB, app.TaskService)
		taskRoutes := protected.Group("/tasks")
		{
			taskRoutes.POST("", taskHandler.CreateTask)
			taskRoutes.PUT("/:id", taskHandler.UpdateTask)
			taskRoutes.DELETE("/:id", taskHandler.DeleteTask)
			taskRoutes.GET("/:id", taskHandler.GetTaskByID)
			taskRoutes.GET("", taskHandler.GetTasks)
		}

		// User routes
		userHandler := handlers.NewUserHandler(app.DB, app.UserService, app.AuthzService)
		userRoutes := protected.Group("/users")
		{
			userRoutes.DELETE("/:user_id", userHandler.DeleteUser)
			userRoutes.GET("", userHandler.GetUsers)
			userRoutes.GET("/:user_id/tasks", taskHandler.GetTasksByUser)
			userRoutes.GET("/profile", userHandler.GetUserProfile)
			userRoutes.GET("/profile/:user_id", userHandler.GetUserProfileByUserId)
			userRoutes.PUT("/profile", userHandler.UpdateUserProfile)
		}

		// Cache management routes
		if app.Cache != nil {
			cacheHandler := handlers.NewCacheHandler(app.CacheManager, app.Cache)
			cacheRoutes := protected.Group("/cache")
			{
				// Original endpoints
				cacheRoutes.GET("/stats", cacheHandler.GetCacheStats)
				cacheRoutes.DELETE("/clear", app.clearCacheHandler()) // (admin only - add authorization check in handler)

				// Cache warming management endpoints
				cacheRoutes.POST("/warm", cacheHandler.WarmCache)
				cacheRoutes.GET("/health", cacheHandler.GetCacheHealth)

				// Job management endpoints
				jobRoutes := cacheRoutes.Group("/jobs")
				{
					jobRoutes.POST("/warmup", cacheHandler.EnqueueWarmupJob)
					jobRoutes.POST("/batch", cacheHandler.EnqueueBatchJob)
					jobRoutes.POST("/scheduled", cacheHandler.EnqueueScheduledJob)
					jobRoutes.DELETE("/evict/:key", cacheHandler.EvictCacheKey)
				}
			}
		}
	}

	app.Router = r
}

func (app *Application) startServer() {
	addr := app.Config.GetServerAddr()

	app.Server = &http.Server{
		Addr:         addr,
		Handler:      app.Router,
		ReadTimeout:  app.Config.Server.ReadTimeout,
		WriteTimeout: app.Config.Server.WriteTimeout,
		IdleTimeout:  app.Config.Server.IdleTimeout,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("🛑 Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.Server.Shutdown(ctx); err != nil {
			log.Printf("❌ Server forced to shutdown: %v", err)
		}

		app.cleanup()
		log.Println("✅ Server stopped gracefully")
	}()

	log.Printf("🚀 Server starting on %s", addr)
	log.Printf("📊 Metrics available at http://%s/metrics", addr)
	log.Printf("💚 Health check at http://%s/health", addr)

	if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

func (app *Application) cleanup() {
	log.Println("🧹 Cleaning up resources...")

	if app.CacheManager != nil {
		if err := app.CacheManager.Stop(); err != nil {
			log.Printf("⚠️  Error stopping cache manager: %v", err)
		}
	}

	if app.Cache != nil {
		if err := app.Cache.Close(); err != nil {
			log.Printf("⚠️  Error closing cache: %v", err)
		}
	}

	if app.Redis != nil {
		if err := app.Redis.Close(); err != nil {
			log.Printf("⚠️  Error closing Redis: %v", err)
		}
	}

	if app.DB != nil {
		sqlDB, err := app.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("⚠️  Error closing database: %v", err)
			}
		}
	}

	log.Println("✅ Cleanup complete")
}

func (app *Application) healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "task-manager-backend",
		}

		sqlDB, err := app.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			health["status"] = "unhealthy"
			health["database"] = "down"
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		health["database"] = "up"

		if app.Redis != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := app.Redis.Ping(ctx).Err(); err != nil {
				health["redis"] = "down"
			} else {
				health["redis"] = "up"
			}
		}

		c.JSON(http.StatusOK, health)
	}
}

func (app *Application) readinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := app.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ready":  false,
				"reason": "database not ready",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ready": true,
		})
	}
}

func (app *Application) cacheStatsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := app.Cache.Stats()
		c.JSON(http.StatusOK, stats)
	}
}

func (app *Application) clearCacheHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User ID not found in context",
			})
			return
		}

		var userIDParsed string
		switch v := userIDStr.(type) {
		case string:
			userIDParsed = v
		default:
			userIDParsed = fmt.Sprintf("%v", v)
		}

		userID, err := uuid.FromString(userIDParsed)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_user_id",
				"message": "Invalid user ID format",
			})
			return
		}

		ctx := c.Request.Context()
		isAdmin, err := app.AuthzService.HasRole(ctx, userID, "admin")
		if err != nil {
			log.Printf("Error checking admin role: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "authorization_failed",
				"message": "Failed to verify admin role",
			})
			return
		}

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Admin role required to clear cache",
			})
			return
		}

		if err := app.Cache.DeletePattern("*"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Cache cleared by admin user: %s", userID)
		c.JSON(http.StatusOK, gin.H{"message": "cache cleared successfully"})
	}
}
