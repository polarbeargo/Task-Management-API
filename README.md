# Task Management API

A enterprise-grade Task Management API, featuring advanced caching, authentication, authorization, and monitoring capabilities. Startar code from [Udacity Taskify : Task Management API](https://github.com/udacity/cd14130-starter/tree/main/starter).

## Software Stack

- **Language**: Go 1.23
- **Web Framework**: Gin
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **ORM**: GORM
- **Authentication**: JWT
- **Testing**: Go built-in testing + Testify
- **Containerization**: Docker & Docker Compose
- **Monitoring**: Prometheus & Grafana. 

## Quick Start
### Automated Setup

```bash
git clone https://github.com/polarbeargo/Task-Management-API.git
cd Task-Management-API

./setup-scalable.sh

./run-prod.sh
./test-scaling.sh
```
Details on configuration and environment variables can be found in the [SET_UP&RUN_DEMO.md](SET_UP.md) file.

## System Architecture

### High-Level Architecture Overview

```mermaid
graph TB
    subgraph "Client Layer"
        WEB[Web Frontend]
        MOBILE[Mobile App]
        API_CLIENT[API Clients]
    end

    subgraph "API Gateway & Load Balancer"
        LB[Nginx Load Balancer]
    end

    subgraph "Global Middleware Stack"
        LOGGER[Logger]
        GIN_REC[Gin Recovery]
        MET_MW[Metrics Middleware]
        REC[Recovery With Log]
        SEC[Security Headers]
        RATE[Rate Limiter]
        CORS[CORS]
    end

    subgraph "Application Layer"
        subgraph "HTTP Handlers"
            TH[Task Handler]
            AH[Auth Handler]
            RH[Register Handler]
            REF[Refresh Handler]
            UH[User Handler]
            CH[Cache Handler]
        end
        
        subgraph "Route Middleware"
            AUTH[Auth Middleware]
            ADMIN[Admin Only Middleware]
        end
        
        subgraph "Business Services"
            TS[Task Service]
            CTS[Cached Task Service]
            AS[Auth Service]
            RS[Register Service]
            US[User Service]
            AUTHZS[Authorization Service]
        end
    end

    subgraph "Caching Layer"
        subgraph "Multi-Level Cache System"
            UCM[Unified Cache Manager]
            L1[L1: Memory Cache]
            L2[L2: Redis Cache]
            CB[Circuit Breaker]
            ICW[Integrated Cache Warmer]
            CM[Cache Metrics]
        end
    end

    subgraph "Data Layer"
        DB[(PostgreSQL)]
        REDIS[(Redis)]
    end

    subgraph "Monitoring & Observability"
        PROM[Prometheus Metrics]
        HEALTH[Health Checks]
        LOGS[Structured Logging]
    end

    %% Client to Load Balancer
    WEB --> LB
    MOBILE --> LB
    API_CLIENT --> LB
    
    %% Load Balancer to Middleware Chain (Sequential)
    LB --> LOGGER
    LOGGER --> GIN_REC
    GIN_REC --> MET_MW
    MET_MW --> REC
    REC --> SEC
    SEC --> RATE
    RATE --> CORS
    
    %% Middleware to Handlers
    CORS --> TH
    CORS --> AH
    CORS --> RH
    CORS --> REF
    CORS --> UH
    CORS --> CH
    
    %% Handlers to Route Middleware
    TH --> AUTH
    UH --> AUTH
    CH --> AUTH
    AUTH --> ADMIN
    
    %% Handlers to Services
    TH --> CTS
    AH --> AS
    RH --> RS
    REF --> AS
    UH --> US
    UH --> AUTHZS
    CH --> UCM
    
    %% Service Layer Interactions
    CTS --> TS
    CTS --> UCM
    TS --> AUTHZS
    
    %% Cache Architecture
    UCM --> L1
    UCM --> L2
    UCM --> ICW
    L1 --> CM
    L2 --> CB
    CB --> REDIS
    ICW --> L1
    ICW --> REDIS
    
    %% Database Connections
    TS --> DB
    AS --> DB
    RS --> DB
    US --> DB
    AUTHZS --> DB
    
    %% Monitoring Connections
    MET_MW --> PROM
    CM --> PROM
    HEALTH --> DB
    HEALTH --> REDIS
    REC --> LOGS
    CB --> LOGS

    %% Styling with Better Colors
    classDef clientClass fill:#4A90E2,stroke:#2E5C8A,stroke-width:2px,color:#fff
    classDef gatewayClass fill:#7B68EE,stroke:#5A4CAD,stroke-width:2px,color:#fff
    classDef middlewareClass fill:#50C878,stroke:#3A9B5C,stroke-width:2px,color:#fff
    classDef handlerClass fill:#FF6B6B,stroke:#CC5555,stroke-width:2px,color:#fff
    classDef authClass fill:#FFA500,stroke:#CC8400,stroke-width:2px,color:#fff
    classDef serviceClass fill:#20B2AA,stroke:#188F89,stroke-width:2px,color:#fff
    classDef cacheClass fill:#FFD700,stroke:#CCB000,stroke-width:2px,color:#000
    classDef dataClass fill:#9370DB,stroke:#7158B0,stroke-width:2px,color:#fff
    classDef monitorClass fill:#32CD32,stroke:#28A428,stroke-width:2px,color:#fff

    class WEB,MOBILE,API_CLIENT clientClass
    class LB gatewayClass
    class LOGGER,GIN_REC,MET_MW,REC,SEC,RATE,CORS middlewareClass
    class TH,AH,RH,REF,UH,CH handlerClass
    class AUTH,ADMIN authClass
    class TS,CTS,AS,RS,US,AUTHZS serviceClass
    class UCM,L1,L2,CB,ICW,CM cacheClass
    class DB,REDIS dataClass
    class PROM,HEALTH,LOGS monitorClass
```

## Class Diagrams

### Core Domain Models

```mermaid
classDiagram
    class User {
        +UUID ID
        +string Username
        +string Email
        +string Password
        +string FirstName
        +string LastName
        +string Department
        +string Position
        +bool IsActive
        +*time.Time LastLoginAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +Role[] Roles
        +UserRole[] UserRoles
        +UserAttribute[] Attributes
        +Task[] Tasks
        +AuditLog[] AuditLogs
        +HasRole(roleName string) bool
        +HasPermission(resource, action string) bool
        +IsAdmin() bool
        +IsUser() bool
        +GetRoleNames() []string
        +GetPermissions() []Permission
    }

    class Task {
        +UUID ID
        +UUID UserID
        +string Title
        +string Description
        +string Status
        +time.Time DueDate
        +string Priority
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
    }

    class Role {
        +UUID ID
        +string Name
        +string Description
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +UUID CreatedBy
        +UUID ModifiedBy
        +UserRole[] Users
        +RolePermission[] Permissions
    }

    class Permission {
        +UUID ID
        +string Name
        +string Resource
        +string Action
        +string Description
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +UUID CreatedBy
        +UUID ModifiedBy
        +RolePermission[] Roles
    }

    class Token {
        +UUID ID
        +UUID UserId
        +UUID JTI
        +string RefreshToken
        +time.Time ExpiresAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
    }

    class UserRole {
        +UUID ID
        +UUID UserID
        +UUID RoleID
        +UUID AssignedBy
        +time.Time AssignedAt
        +*time.Time ExpiresAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +User User
        +Role Role
        +User AssignedByUser
        +IsExpired() bool
    }

    class RolePermission {
        +UUID ID
        +UUID RoleID
        +UUID PermissionID
        +UUID AssignedBy
        +time.Time AssignedAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +Role Role
        +Permission Permission
        +User AssignedByUser
    }

    class UserAttribute {
        +UUID ID
        +UUID UserID
        +string Name
        +string Value
        +string Type
        +string Source
        +*time.Time ExpiresAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +User User
        +IsExpired() bool
        +GetTypedValue() any
    }

    class ResourceAttribute {
        +UUID ID
        +string ResourceType
        +UUID ResourceID
        +string Name
        +string Value
        +string Type
        +string Source
        +*time.Time ExpiresAt
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +gorm.DeletedAt DeletedAt
        +IsExpired() bool
        +GetTypedValue() any
    }

    class AuditLog {
        +UUID ID
        +UUID UserID
        +string Action
        +string Resource
        +UUID ResourceID
        +string Decision
        +string Reason
        +string IPAddress
        +string UserAgent
        +string RequestMethod
        +string RequestPath
        +string Context
        +time.Time Timestamp
        +User User
    }

    class AuthorizationRequest {
        +UUID UserID
        +string Action
        +string Resource
        +UUID ResourceID
        +map[string]string UserAttributes
        +map[string]string ResourceAttributes
        +map[string]any Context
    }

    class AuthorizationDecision {
        +string Decision
        +string Reason
    }

    %% Relationships
    User --> Task
    User --> UserRole
    User --> Token
    User --> UserAttribute
    User --> AuditLog
    
    Role --> UserRole
    Role --> RolePermission
    
    Permission --> RolePermission
    
    UserRole --> User
    UserRole --> Role
    
    RolePermission --> Role
    RolePermission --> Permission
    RolePermission --> User
    
    UserAttribute --> User
    
    AuditLog --> User

```

### Service Layer Architecture
A clean, interface-driven design that separates business logic from infrastructure concerns. 
- Advanced Design Patterns
    1. Decorator Pattern ([CachedTaskService](https://github.com/polarbeargo/Task-Management-API/blob/b8202d94082ef756acef656e5f2296c0aaf40ae7/backend/internal/services/cached_tasks.go#L15))
    2. Strategy Pattern ([ABAC Evaluation](https://github.com/polarbeargo/Task-Management-API/blob/b8202d94082ef756acef656e5f2296c0aaf40ae7/backend/internal/services/authorization.go#L192))
    3. Default Composition Over Inheritance 
```mermaid
classDiagram
    class TaskService {
        <<interface>>
        +CreateTask(db *gorm.DB, task models.Task) error
        +GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error)
        +GetTasks(db *gorm.DB) ([]models.Task, error)
        +UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error
        +DeleteTask(db *gorm.DB, id uuid.UUID) error
        +GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error)
    }

    class TaskServiceImpl {
        +CreateTask(db *gorm.DB, task models.Task) error
        +GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error)
        +GetTasks(db *gorm.DB) ([]models.Task, error)
        +UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error
        +DeleteTask(db *gorm.DB, id uuid.UUID) error
        +GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error)
    }

    class CachedTaskService {
        -taskService TaskService
        -cache *MultiLevelCache
        -warmingActive bool
        +CreateTask(db *gorm.DB, task models.Task) error
        +GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error)
        +GetTasks(db *gorm.DB) ([]models.Task, error)
        +UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error
        +DeleteTask(db *gorm.DB, id uuid.UUID) error
        +GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error)
        +GetTasksByUser(db *gorm.DB, userID uuid.UUID) ([]models.Task, error)
        +GetCacheStats() map[string]any
        +StartCacheWarming(ctx context.Context)
        +StopCacheWarming()
        +WarmCriticalData(ctx context.Context, db *gorm.DB) error
        -setupCacheWarming()
    }

    class AuthService {
        <<interface>>
        +LoginUser(db *gorm.DB, email, password string) (*models.User, error)
        +GenerateToken(db *gorm.DB, userID uuid.UUID) (string, string, error)
        +RefreshToken(db *gorm.DB, refreshToken string) (string, string, int64, error)
        +GetUserPermissions(db *gorm.DB, userID uuid.UUID) ([]string, error)
        +RevokeToken(db *gorm.DB, refreshToken string) error
    }

    class AuthServiceImpl {
        +LoginUser(db *gorm.DB, email, password string) (*models.User, error)
        +GenerateToken(db *gorm.DB, userID uuid.UUID) (string, string, error)
        +RefreshToken(db *gorm.DB, refreshToken string) (string, string, int64, error)
        +GetUserPermissions(db *gorm.DB, userID uuid.UUID) ([]string, error)
        +RevokeToken(db *gorm.DB, refreshToken string) error
    }

    class UserService {
        <<interface>>
        +GetUserProfile(db *gorm.DB, userID uuid.UUID) (models.User, error)
        +GetUserProfileMalicious(db *gorm.DB, userID string) ([]models.User, error)
        +GetUsers(db *gorm.DB) ([]models.User, error)
        +DeleteUser(db *gorm.DB, userId uuid.UUID) error
    }

    class UserServiceImpl {
        +GetUserProfile(db *gorm.DB, userID uuid.UUID) (models.User, error)
        +GetUserProfileMalicious(db *gorm.DB, userID string) ([]models.User, error)
        +GetUsers(db *gorm.DB) ([]models.User, error)
        +DeleteUser(db *gorm.DB, userId uuid.UUID) error
    }

    class RegisterService {
        <<interface>>
        +RegisterUser(db *gorm.DB, req RegistrationRequest) (*models.User, error)
    }

    class RegisterServiceImpl {
        +RegisterUser(db *gorm.DB, req RegistrationRequest) (*models.User, error)
    }

    class AuthorizationService {
        <<interface>>
        +HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
        +HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
        +AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
        +RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
        +IsAuthorized(ctx context.Context, request AuthorizationRequest) (*AuthorizationDecision, error)
        +SetUserAttribute(ctx context.Context, userID uuid.UUID, key, value, dataType string) error
        +SetResourceAttribute(ctx context.Context, resourceType string, resourceID uuid.UUID, key, value, dataType string) error
        +CreateRole(ctx context.Context, name, description string) (*models.Role, error)
        +CreatePermission(ctx context.Context, resource, action, description string) (*models.Permission, error)
        +GrantPermissionToRole(ctx context.Context, roleID, permissionID, grantedBy uuid.UUID) error
        +LogAuthorizationDecision(ctx context.Context, decision AuthorizationDecision) error
    }

    class AuthorizationServiceImpl {
        -db *gorm.DB
        +HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
        +HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
        +AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
        +RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
        +IsAuthorized(ctx context.Context, request AuthorizationRequest) (*AuthorizationDecision, error)
        +SetUserAttribute(ctx context.Context, userID uuid.UUID, key, value, dataType string) error
        +SetResourceAttribute(ctx context.Context, resourceType string, resourceID uuid.UUID, key, value, dataType string) error
        +CreateRole(ctx context.Context, name, description string) (*models.Role, error)
        +CreatePermission(ctx context.Context, resource, action, description string) (*models.Permission, error)
        +GrantPermissionToRole(ctx context.Context, roleID, permissionID, grantedBy uuid.UUID) error
        +LogAuthorizationDecision(ctx context.Context, decision AuthorizationDecision) error
        -evaluateABACPolicies(ctx context.Context, request AuthorizationRequest) (bool, string, error)
        -evaluateTaskABACPolicy(ctx context.Context, request AuthorizationRequest, userAttrs map[string]string) (bool, string, error)
        -evaluateUserABACPolicy(ctx context.Context, request AuthorizationRequest, userAttrs map[string]string) (bool, string, error)
    }

    %% Relationships
    TaskService <|-- TaskServiceImpl : implements
    TaskService <|-- CachedTaskService : implements
    AuthService <|-- AuthServiceImpl : implements
    UserService <|-- UserServiceImpl : implements
    RegisterService <|-- RegisterServiceImpl : implements
    AuthorizationService <|-- AuthorizationServiceImpl : implements
    
    CachedTaskService --> TaskServiceImpl : decorates
    CachedTaskService --> MultiLevelCache : uses

    %% Styling
    classDef interfaceClass fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef serviceClass fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    classDef decoratorClass fill:#fff3e0,stroke:#e65100,stroke-width:2px

```

### Multi-Level Caching System - Multi-Level Intelligence 

- **Cache Interface:**
    - One interface, multiple implementations - clean abstraction for flexibility.
    - **Implementations:**
        - **MemoryCache**: In-memory L1 cache with sync.Map for concurrent access
        - **RedisCache**: Distributed L2 cache with connection pooling
        - **MultiLevelCache**: Composite implementation orchestrating L1 + L2

```mermaid
classDiagram
    class Cache {
        <<interface>>
        +Set(key string, value any, ttl Duration) error
        +Get(key string, dest any) error
        +Delete(key string) error
        +DeletePattern(pattern string) error
        +Exists(key string) (bool, error)
        +Stats() map[string]any
        +Health() error
        +Close() error
    }

    class MultiLevelCache {
        -l1 *MemoryCache
        -l2 *RedisCache
        -metrics *CacheMetrics
        -circuitBreaker *CircuitBreaker
        -warmer *CacheWarmer
        +Set(key string, value any, ttl Duration) error
        +Get(key string, dest any) error
        +Delete(key string) error
        +DeletePattern(pattern string) error
        +Exists(key string) (bool, error)
        +Stats() map[string]any
        +Health() error
        +Close() error
        +GetWarmer() *CacheWarmer
        +GetMetrics() *CacheMetrics
        +GetCircuitBreaker() *CircuitBreaker
    }

    class MemoryCache {
        -store sync.Map
        -mutex sync.RWMutex
        +Set(key string, value any, ttl Duration) error
        +Get(key string) (any, bool)
        +Delete(key string) error
        +DeletePattern(pattern string) error
        +Exists(key string) (bool, error)
        +Clear() error
        +Stats() map[string]any
        +Close() error
        -cleanup()
    }

    class RedisCache {
        -client *redis.Client
        -ctx context.Context
        +Set(key string, value any, ttl Duration) error
        +Get(key string, dest any) error
        +Delete(key string) error
        +DeletePattern(pattern string) error
        +Exists(key string) (bool, error)
        +SetWithTags(key string, value any, ttl Duration, tags []string) error
        +InvalidateByTag(tag string) error
        +Health() error
        +Stats() map[string]any
        +Close() error
    }

    class CacheMetrics {
        +Hits int64
        +Misses int64
        +Errors int64
        +Sets int64
        +Deletes int64
        +StartTime int64
        +RecordHit()
        +RecordMiss()
        +RecordSet()
        +RecordDelete()
        +RecordError()
        +GetStats() CacheMetrics
        +HitRate() float64
        +Reset()
    }

    class CircuitBreaker {
        -mu sync.RWMutex
        -state CircuitBreakerState
        -failureCount int
        -successCount int
        -lastFailureTime time.Time
        -maxFailures int
        -timeout time.Duration
        -halfOpenMaxCalls int
        +Execute(operation func() error) error
        +GetState() CircuitBreakerState
        +GetStats() map[string]any
        -allow() bool
        -shouldAttemptReset() bool
        -recordSuccess()
        -recordFailure()
    }

    class CacheWarmer {
        -cache Cache
        -strategy *WarmupStrategy
        -mu sync.RWMutex
        -running bool
        -stopCh chan bool
        -workerPool *WorkerPool
        -scheduler *JobScheduler
        -priorityQueue *PriorityQueue
        +AddWarmupJob(job WarmupJob)
        +Start()
        +Stop()
        +WarmCache() error
        +GetStats() map[string]any
        +IsRunning() bool
        -shouldWarmup() bool
        -processJobs(jobs []WarmupJob)
        -executeBatch(batch []WarmupJob)
    }

    class WarmupJob {
        +Key string
        +Data any
        +TTL time.Duration
        +Priority int
    }

    %% Relationships
    Cache <|-- MultiLevelCache : implements
    Cache <|-- MemoryCache : implements
    Cache <|-- RedisCache : implements

    MultiLevelCache --> MemoryCache : L1 cache
    MultiLevelCache --> RedisCache : L2 cache
    MultiLevelCache --> CacheMetrics : tracks performance
    MultiLevelCache --> CircuitBreaker : fault tolerance
    MultiLevelCache --> CacheWarmer : preloads data

    CacheWarmer --> WarmupJob : processes

    %% Styling
    classDef interfaceClass fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef cacheClass fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef componentClass fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px

```
- The **Circuit Breaker** implements a **Finite State Machine (FSM)** for enterprise reliability and fault tolerance.
    - **Circuit Breaker States:**
        - **Closed**: Normal operation, all requests pass through.
        - **Open**: Redis is down, skip L2 calls for fast-fail.
        - **Half-Open**: Testing recovery, limited calls allowed.  
    - **State Transition Diagram:**

```mermaid
stateDiagram-v2
    [*] --> CLOSED
    CLOSED --> OPEN: Failures >= 5
    OPEN --> HALF_OPEN: Timeout (30s)
    HALF_OPEN --> CLOSED: Success Count >= 3
    HALF_OPEN --> OPEN: Any Failure
    CLOSED --> CLOSED: Success (reset counter)
```

## Cache Warming Job/Worker System
`UnifiedCacheManager` is the brain with [auto-detection logic](https://github.com/polarbeargo/Task-Management-API/blob/07064a662b72e0dc3288d70042c22444ea4f96c3/backend/main.go#L136) that automatically chooses optimal mode based on Redis availability and system configuration.
- Dual-Mode Operation
    - Integrated Mode: Redis + Distributed Workers + Job Routing
    - Legacy Mode: Local Workers + Priority Queues + Fallback Safety


```mermaid
classDiagram
    class UnifiedCacheManager {
        -IntegratedCacheWarmer* integratedWarmer
        -CacheWarmer* legacyWarmer
        -bool useIntegrated
        -context.Context ctx
        +NewUnifiedCacheManager(cache Cache, config *CacheWarmingConfig) *UnifiedCacheManager
        +Start() error
        +Stop() error
        +IsRunning() bool
        +IsIntegrated() bool
        +WarmupCache() error
        +EnqueueWarmupJob(key string, data any, ttl Duration, priority int) error
        +EnqueueBatchWarmupJob(keys []string, data any, priority int) error
        +EnqueueScheduledWarmup(key string, data any, ttl Duration, processAt Time, priority int) error
        +EnqueueEvictionJob(key string, priority int) error
        +EnqueueValidationJob(key string, expectedData any, priority int) error
        +GetMetrics() map[string]any
        +GetQueueSizes() (map[string]int64, error)
        +GetSystemInfo() map[string]any
        +HealthCheck() map[string]any
        +WarmupUserData(userID string, userData any, ttl Duration) error
        +WarmupBatchUserData(userIDs []string, userDataMap map[string]any, ttl Duration) error
        +WarmupPopularContent(contentType string, items []any, ttl Duration) error
        +UpdateConfiguration(newStrategy *WarmupStrategy) error
        +SchedulePeriodicWarmup(key string, data any, ttl Duration, interval Duration) error
        -determineMode(config *CacheWarmingConfig, cache Cache) CacheWarmingMode
    }

    class IntegratedCacheWarmer {
        -Cache cache
        -*redis.Client redisClient
        -*WarmupStrategy strategy
        -sync.RWMutex mu
        -bool running
        -chan stopCh
        -[]*DistributedCacheWorker distributedWorkers
        -*WorkerPool localWorkerPool
        -*JobRouter jobRouter
        -*JobScheduler scheduler
        -*PriorityQueue priorityQueue
        -int64 processedJobs
        -int64 failedJobs
        -int64 retryJobs
        -int64 distributedJobs
        -int64 localJobs
        +NewIntegratedCacheWarmer(cache Cache, redisClient *redis.Client, strategy *WarmupStrategy) *IntegratedCacheWarmer
        +Start() error
        +Stop() error
        +EnqueueWarmupJob(key string, data any, ttl Duration, priority int) error
        +EnqueueBatchWarmupJob(keys []string, data any, priority int) error
        +EnqueueScheduledWarmup(key string, data any, ttl Duration, processAt Time, priority int) error
        +GetMetrics() map[string]any
        +GetQueueSizes() (map[string]int64, error)
        +WarmupCache() error
        +IsRunning() bool
        +routeJob(job DistributedCacheJob) error
        -enqueueToRedis(job DistributedCacheJob) error
        -enqueueToLocal(job DistributedCacheJob) error
    }

    class CacheWarmer {
        -Cache cache
        -*WarmupStrategy strategy
        -sync.RWMutex mu
        -bool running
        -chan stopCh
        -*WorkerPool workerPool
        -*JobScheduler scheduler
        -*PriorityQueue priorityQueue
        +NewCacheWarmer(cache Cache, strategy *WarmupStrategy) *CacheWarmer
        +AddWarmupJob(job WarmupJob)
        +Start(ctx Context)
        +Stop()
        +WarmCacheManually(ctx Context)
        +GetStats() map[string]any
        -warmCache(ctx Context) error
        -processBatch(ctx Context, jobs []WarmupJob, concurrency int) error
        -processJob(job WarmupJob) error
        -shouldWarmup() bool
    }

    class DistributedCacheWorker {
        -string id
        -*IntegratedCacheWarmer warmer
        -*redis.Client redisClient
        -chan stopCh
        -sync.WaitGroup wg
        -string queueName
        +start()
        -processJobs()
        -moveDelayedJobs(ctx Context)
        -processImmediateJobs(ctx Context)
        -executeJob(ctx Context, job *DistributedCacheJob)
        -handleWarmupJob(ctx Context, job *DistributedCacheJob) error
        -handleBatchJob(ctx Context, job *DistributedCacheJob) error
        -handleValidationJob(ctx Context, job *DistributedCacheJob) error
        -handleEvictionJob(ctx Context, job *DistributedCacheJob) error
    }

    class WorkerPool {
        -int workers
        -chan WarmupJob jobCh
        -chan JobResult resultCh
        -Cache cache
        -context.Context ctx
        -context.CancelFunc cancel
        -sync.WaitGroup wg
        -bool running
        -sync.RWMutex mu
        -int64 jobsProcessed
        -time.Duration totalDuration
        -int64 errors
        +NewWorkerPool(workers int, cache Cache) *WorkerPool
        +Start()
        +Stop()
        +SubmitJob(job WarmupJob) bool
        +SubmitJobs(jobs []WarmupJob) int
        +GetStats() map[string]any
        +IsRunning() bool
        +Resize(newSize int)
        -worker(id int)
        -processJob(job WarmupJob) JobResult
        -resultCollector()
    }

    class JobRouter {
        -bool redisAvailable
        -int localCapacity
        -string distributedQueue
        -sync.RWMutex mu
    }

    class JobScheduler {
        -*CacheWarmer warmer
        -*WorkerPool workerPool
        -*PriorityQueue queue
        -map[string]*ScheduleTrigger triggers
        -bool running
        -context.Context ctx
        -context.CancelFunc cancel
        -sync.RWMutex mu
        -time.Time lastRun
        -time.Time nextRun
        -int64 runCount
        +NewJobScheduler(warmer *CacheWarmer, workerPool *WorkerPool) *JobScheduler
        +AddIntervalTrigger(name string, interval Duration, condition func() bool)
        +AddEventTrigger(name string, condition func() bool)
        +RemoveTrigger(name string)
        +ScheduleJob(job WarmupJob)
        +ScheduleJobs(jobs []WarmupJob)
        +Start()
        +Stop()
        +ProcessScheduledJobs() int
        +GetStats() map[string]any
        +IsRunning() bool
        +ClearQueue()
        -schedulerLoop()
        -checkTriggers()
        -extractJobsBatch() []WarmupJob
    }

    class PriorityQueue {
        -[]*JobQueue items
        -sync.RWMutex mu
        +NewPriorityQueue() *PriorityQueue
        +Push(job WarmupJob)
        +Pop() (WarmupJob, bool)
        +Peek() (WarmupJob, bool)
        +Len() int
        +Empty() bool
        +Clear()
        +GetJobs() []WarmupJob
    }

    class WarmupJob {
        +string Key
        +any Data
        +time.Duration TTL
        +int Priority
    }

    class DistributedCacheJob {
        +string ID
        +CacheJobType Type
        +map[string]any Payload
        +int Attempts
        +int MaxTries
        +time.Time CreatedAt
        +time.Time ProcessAt
        +int Priority
    }

    class WarmupStrategy {
        +[]WarmupJob Jobs
        +int BatchSize
        +int ConcurrentJobs
        +time.Duration WarmupInterval
        +func() bool HealthCheckFunc
        +bool UseWorkerPool
        +bool UseScheduler
    }

    class CacheWarmingConfig {
        +CacheWarmingMode Mode
        +*redis.Client RedisClient
        +*WarmupStrategy Strategy
        +bool EnableMetrics
        +bool PreferDistributed
        +bool LocalFallback
        +int DistributedThreshold
        +int MaxRetries
    }

    class JobQueue {
        +WarmupJob Job
        +int Priority
        +int Index
    }

    class JobResult {
        +WarmupJob Job
        +error Error
        +time.Duration Duration
    }

    class ScheduleTrigger {
        +string Type
        +time.Duration Interval
        +func() bool Condition
    }

    class PriorityQueueHeap {
        <<interface>>
        +Len() int
        +Less(i, j int) bool
        +Swap(i, j int)
        +Push(x any)
        +Pop() any
    }

    %% Enums and Constants
    class CacheJobType {
        <<enumeration>>
        CacheJobWarmup
        CacheJobBatch
        CacheJobEviction
        CacheJobValidation
        CacheJobScheduled
    }

    class CacheWarmingMode {
        <<enumeration>>
        ModeAuto
        ModeLegacy
        ModeIntegrated
        ModeLocalOnly
        ModeDistributed
    }

    %% Relationships
    UnifiedCacheManager --> IntegratedCacheWarmer : manages (integrated mode)
    UnifiedCacheManager --> CacheWarmer : manages (legacy mode)
    UnifiedCacheManager --> CacheWarmingConfig : configured by
    
    IntegratedCacheWarmer --> DistributedCacheWorker : contains multiple
    IntegratedCacheWarmer --> WorkerPool : uses (local processing)
    IntegratedCacheWarmer --> JobRouter : routes jobs with
    IntegratedCacheWarmer --> JobScheduler : schedules with
    IntegratedCacheWarmer --> PriorityQueue : queues jobs in
    IntegratedCacheWarmer --> WarmupStrategy : configured by
    IntegratedCacheWarmer --> DistributedCacheJob : processes
    
    CacheWarmer --> WorkerPool : manages
    CacheWarmer --> JobScheduler : coordinates
    CacheWarmer --> PriorityQueue : queues jobs
    CacheWarmer --> WarmupStrategy : uses
    CacheWarmer --> WarmupJob : processes
    
    DistributedCacheWorker --> DistributedCacheJob : processes
    DistributedCacheWorker --> IntegratedCacheWarmer : reports to
    
    WorkerPool --> WarmupJob : processes
    WorkerPool --> JobResult : produces
    
    JobScheduler --> ScheduleTrigger : uses
    JobScheduler --> CacheWarmer : triggers
    JobScheduler --> WorkerPool : submits to
    JobScheduler --> PriorityQueue : manages
    
    PriorityQueue --> JobQueue : contains
    PriorityQueue --> WarmupJob : stores
    PriorityQueueHeap --> JobQueue : heap operations
    
    JobQueue --> WarmupJob : wraps
    WarmupStrategy --> WarmupJob : contains
    DistributedCacheJob --> CacheJobType : typed by
    CacheWarmingConfig --> CacheWarmingMode : configured by

    %% Styling
    classDef managerClass fill:#e1f5fe,stroke:#03a9f4,stroke-width:2px
    classDef workerClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    classDef jobClass fill:#e8f5e8,stroke:#4caf50,stroke-width:2px
    classDef componentClass fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px
    classDef configClass fill:#fce4ec,stroke:#e91e63,stroke-width:2px
    classDef interfaceClass fill:#fff8e1,stroke:#ffc107,stroke-width:2px
    classDef enumClass fill:#f1f8e9,stroke:#8bc34a,stroke-width:2px
    
``` 
### Cache Job Types: Complete Lifecycle Management
The unified cache management system supports five intelligent job types for cache lifecycle control:

| Job Type | API Function | Priority Range | Description | Use Cases |
|----------|-------------|----------------|-------------|-----------|
| **1. Warmup Jobs** | `EnqueueWarmupJob(key, data, ttl, priority)` | 1-10 (1=immediate) | Core intelligence for individual cache entries with smart TTL management. Priority-based execution with automatic retry logic and exponential backoff. | • Pre-load user profiles on login<br/>• Cache critical configuration data<br/>• Warm frequently accessed resources |
| **2. Batch Jobs** | `EnqueueBatchWarmupJob(keys, data, priority)` | 1-10 | Efficiency at scale for bulk operations on related data sets. Atomic processing ensures consistency with reduced network overhead for better performance. | • Cache entire user session data<br/>• Pre-warm product catalog pages<br/>• Bulk import initial data sets |
| **3. Scheduled Jobs** | `EnqueueScheduledWarmup(key, data, ttl, processAt, priority)` | 1-10 | Time-based execution using Unix timestamp scoring. Perfect for cache refresh cycles and maintenance windows with delayed queue management and automatic promotion. | • Scheduled nightly cache refresh<br/>• Pre-warm before traffic spikes<br/>• Periodic data synchronization |
| **4. Validation Jobs** | `EnqueueValidationJob(key, expectedData, priority)` | 1-10 | Data integrity checks against expected values. Cache consistency verification with automatic re-warming when validation fails. Self-healing capabilities for corrupted data. | • Verify critical cache accuracy<br/>• Detect cache corruption<br/>• Ensure data consistency |
| **5. Eviction Jobs** | `EnqueueEvictionJob(key, priority)` | 1-10 | Controlled cache invalidation for outdated data. Pattern-based cleanup for related entries with memory optimization and capacity management. | • Clear user session on logout<br/>• Invalidate stale product data<br/>• Memory pressure management |

**Key Features:**
- **Priority-Based Execution**: Priority 1 executes immediately (high-priority critical data), priority 10 runs in background (low-priority bulk operations)
- **Smart Retry Logic**: Automatic linear backoff for failed operations (1s, 2s, 3s, 4s...)
- **Best-Effort Batch Operations**: Batch jobs process multiple keys efficiently with per-key error handling
- **Pattern Matching**: Eviction supports wildcards (e.g., `user:*`, `session:*`) for related entry cleanup
- **Self-Healing**: Validation jobs automatically re-warm corrupted or missing data
- **Hybrid Processing**: Jobs route intelligently between local in-memory workers (fast) and distributed Redis workers (scalable)
- **Dead Letter Queue**: Failed jobs after max retries (default: 3) move to DLQ for analysis

**Architecture:**
- **5 Distributed Redis Workers**: Process jobs from shared Redis queues (immediate, delayed, DLQ)
- **5 In-Memory Worker Pool**: High-speed local processing for priority 1 jobs
- **Job Router**: Intelligently routes jobs based on priority, queue capacity, and Redis availability
- **Circuit Breaker**: Automatic fallback when Redis is unavailable

### Intelligent Job Routing System

The system uses a **JobRouter** that dynamically routes cache jobs between local in-memory workers and distributed Redis workers based on multiple factors:

#### JobRouter Decision Matrix

The router evaluates jobs in this priority order:

| Priority | Scenario | Routing Decision | Latency | Reason |
|----------|----------|------------------|---------|--------|
| **1st** | **Redis Unavailable** | Local Worker Pool (Fallback) | < 1ms | Graceful degradation, system continues operating without Redis |
| **2nd** | **Scheduled Job Type** | Distributed Redis Queue | ~5-10ms | Time-based execution requires Redis persistence and delayed processing |
| **3rd** | **Priority = 1 (Critical)** | Local Worker Pool | < 1ms | Immediate execution required, bypass network overhead for fastest response |
| **4th** | **Local Queue Full** | Distributed Redis Queue | ~5ms | Prevent blocking when local capacity reached, maintain throughput |
| **5th** | **Normal Priority (2-10)** | Local Worker Pool (Default) | < 1ms | Fast local processing when capacity available, distributed as fallback |

**Routing Logic:**

```mermaid
flowchart TD
    Start([New Cache Job]) --> CheckPriority{Priority = 1?}
    
    CheckPriority -->|Yes| CheckCapacity{Local Queue<br/>Has Capacity?}
    CheckPriority -->|No<br/>Priority 2-10| CheckRedis{Redis<br/>Available?}
    
    CheckCapacity -->|Yes| LocalFast[Route to Local Worker Pool<br/> < 1ms latency<br/>Fastest Path]
    CheckCapacity -->|No<br/>Queue Full| CheckRedis
    
    CheckRedis -->|Yes| DistributedPath[Route to Distributed Redis Queue<br/> ~5-10ms latency<br/>Scalable Path]
    CheckRedis -->|No| LocalFallback[Route to Local Worker Pool<br/> Fallback Mode<br/>< 1ms latency]
    
    LocalFast --> End([Job Enqueued])
    DistributedPath --> End
    LocalFallback --> End
    
    style LocalFast fill:#90EE90,stroke:#2d7a2d,stroke-width:3px,color:#000
    style DistributedPath fill:#87CEEB,stroke:#1e5a8e,stroke-width:3px,color:#000
    style LocalFallback fill:#FFD700,stroke:#b8860b,stroke-width:3px,color:#000
    style CheckPriority fill:#FFE4B5,stroke:#d4a373,stroke-width:2px,color:#000
    style CheckCapacity fill:#FFE4B5,stroke:#d4a373,stroke-width:2px,color:#000
    style CheckRedis fill:#FFE4B5,stroke:#d4a373,stroke-width:2px,color:#000
    style Start fill:#f0f0f0,stroke:#666,stroke-width:2px,color:#000
    style End fill:#f0f0f0,stroke:#666,stroke-width:2px,color:#000
```

**Performance Characteristics:**
- **Local Processing**: Sub-millisecond execution, ideal for critical/time-sensitive jobs
- **Distributed Processing**: 5-10ms latency, excellent for normal priority and bulk operations
- **Automatic Failover**: Seamless transition between modes based on system health
- **Load Balancing**: Prevents local worker saturation by offloading to Redis when needed

**The result:** A cache warming system that doesn't just prevent cold starts—it anticipates user needs, scales automatically, and self-heals to deliver consistently exceptional performance across distributed environments. This system represents a paradigm shift from reactive cache management to predictive performance engineering, ensuring our applications maintain lightning-fast response times 24/7.  
## Sequence Diagrams

### Task Creation Flow with Caching

```mermaid
sequenceDiagram
    participant C as Client
    participant H as TaskHandler
    participant M as AuthzMiddleware
    participant CS as CachedTaskService
    participant TS as TaskServiceImpl
    participant MC as MultiLevelCache
    participant Met as CacheMetrics
    participant L1 as MemoryCache
    participant CB as CircuitBreaker
    participant L2 as RedisCache
    participant DB as Database

    %% Authentication & Authorization
    C->>H: POST /tasks {title, description, status}
    H->>M: AuthzMiddleware validates JWT token
    M->>M: Extract Bearer token & validate
    M->>M: Check user_id, role, permissions from JWT claims
    M-->>H: Authorization granted (sets user context)
    
    %% Request Processing
    H->>H: Get user_id from context (string)
    H->>H: Validate JSON input & bind to taskInput struct
    H->>H: Generate new UUID for task
    H->>H: Convert user_id string to UUID
    H->>H: Create models.Task{ID, UserID, Title, Description, Status}
    H->>CS: CreateTask(db, task)
    
    %% Database Creation
    CS->>TS: CreateTask(db, task)
    TS->>DB: INSERT INTO tasks (id, user_id, title, description, status, created_at, updated_at)
    
    alt Task Creation Successful
        DB-->>TS: Task created successfully
        TS-->>CS: Created task with generated ID
        
        %% Immediate Cache Population
        CS->>CS: Generate cache key: "task:{task.ID}"
        CS->>MC: Set(cacheKey, task, 30min TTL)
        
        %% Multi-Level Cache Set
        MC->>L1: Set(cacheKey, task, 30min TTL)
        L1-->>MC: Stored in memory
        MC->>Met: RecordSet()
        
        alt Redis Available (L2)
            MC->>CB: Execute(L2.Set operation)
            CB->>L2: Set(cacheKey, task, 30min TTL)
            L2-->>CB: Stored in Redis
            CB-->>MC: L2 cache updated
        else Redis Unavailable
            CB->>Met: RecordError()
            note over MC: Task cached in L1 only
        end
        
        %% Cache Invalidation (COMPREHENSIVE)
        note over CS: Invalidate affected list caches
        CS->>MC: DeletePattern("user_tasks:{userID}:*")
        MC->>L1: DeletePattern("user_tasks:{userID}:*")
        MC->>L2: DeletePattern("user_tasks:{userID}:*")
        note over MC: DeletePattern goes directly to L2 (no circuit breaker)
        
        CS->>MC: DeletePattern("tasks_paginated:*")
        MC->>L1: DeletePattern("tasks_paginated:*")
        MC->>L2: DeletePattern("tasks_paginated:*")
        
        CS->>MC: Delete("all_tasks")
        MC->>L1: Delete("all_tasks")
        MC->>Met: RecordDelete()
        
        alt L2 Delete via Circuit Breaker
            MC->>CB: Execute(L2.Delete operation)
            
            alt L2 Delete Successful
                CB->>L2: Delete("all_tasks")
                L2-->>CB: Delete successful
                CB-->>MC: err = nil
            else L2 Delete Failed
                CB->>L2: Delete("all_tasks")
                L2-->>CB: Redis error
                CB-->>MC: err != nil
                MC->>Met: RecordError()
                note over MC: L1 deleted, L2 delete failed
            end
        end
        
        note over CS: Ensures new task appears in all listings immediately
        
        CS-->>H: Task creation successful
        H-->>C: 201 Created {task}
        
    else Task Creation Failed
        DB-->>TS: Error (constraint violation, etc.)
        TS-->>CS: Database error
        CS-->>H: Creation failed
        H->>H: handleTaskError(c, err)
        H-->>C: 400/500 Error response
    end

    %% Error Handling Scenarios
    alt Invalid Input
        H->>H: ShouldBindJSON() validation fails
        H-->>C: 400 Bad Request {error}
    else Authorization Failed
        M-->>C: 401 Unauthorized / 403 Forbidden
    end
```

### Task Retrieval with Multi-Level Cache

```mermaid
sequenceDiagram
    participant C as Client
    participant H as TaskHandler
    participant CS as CachedTaskService
    participant MC as MultiLevelCache
    participant M as CacheMetrics
    participant L1 as MemoryCache
    participant CB as CircuitBreaker
    participant L2 as RedisCache
    participant TS as TaskServiceImpl
    participant DB as Database

    C->>H: GET /tasks/:id
    H->>CS: GetTaskByID(db, id)
    
    CS->>CS: Generate cache key: "task:{id}"
    CS->>MC: Get(cacheKey, &cachedTask)
    
    %% L1 Cache Check
    MC->>L1: Get(cacheKey)
    
    alt Cache Hit (L1 Memory)
        L1-->>MC: Task found in memory (raw interface{})
        MC->>MC: copyValue(L1_value, &cachedTask) via JSON marshal/unmarshal
        MC->>M: RecordHit()
        MC-->>CS: Task from L1 memory
        CS-->>H: Cached task
        H-->>C: 200 OK with task
        
    else Cache Miss (L1 Memory)
        L1-->>MC: Not found in memory
        
        alt L2 Redis Available
            MC->>CB: Execute(L2.Get operation with dest)
            
            alt Circuit Breaker CLOSED
                CB->>L2: Get(cacheKey, &cachedTask)
                
                alt Cache Hit (L2 Redis)
                    L2-->>CB: Task found in Redis (JSON unmarshaled into dest)
                    CB-->>MC: err = nil, l2Hit = true
                    note over MC: Data already in dest (&cachedTask)
                    MC->>L1: Set(cacheKey, dest, 5min TTL)
                    note over L1: Promote to L1 with shorter TTL
                    MC->>M: RecordHit()
                    MC-->>CS: Task from L2 Redis (via dest)
                    CS-->>H: Cached task
                    H-->>C: 200 OK with task
                    
                else Cache Miss (L2 Redis)
                    L2-->>CB: ErrCacheMiss
                    CB-->>MC: err = ErrCacheMiss, l2Hit = false
                    MC->>M: RecordMiss()
                    note over MC: Both L1 and L2 miss recorded
                    
                    %% Database Fallback
                    CS->>TS: GetTaskByID(db, id)
                    TS->>DB: SELECT * FROM tasks WHERE id = ?
                    
                    alt Task Found in DB
                        DB-->>TS: Task record
                        TS-->>CS: Fresh task from DB
                        
                        %% Cache the result
                        CS->>MC: Set(cacheKey, task, 30min TTL)
                        MC->>L1: Set(cacheKey, task, 30min TTL)
                        MC->>M: RecordSet()
                        
                        alt Redis Set via Circuit Breaker
                            MC->>CB: Execute(L2.Set operation)
                            
                            alt L2 Set Successful
                                CB->>L2: Set(cacheKey, task, 30min TTL)
                                L2-->>CB: Set successful
                                CB-->>MC: err = nil
                            else L2 Set Failed
                                CB->>L2: Set(cacheKey, task, 30min TTL)
                                L2-->>CB: Redis error
                                CB->>CB: Record failure, may open circuit
                                CB-->>MC: err != nil
                                MC->>M: RecordError()
                                note over MC: L2 set failed but L1 cached
                            end
                        end
                        
                        CS-->>H: Fresh task from DB
                        H-->>C: 200 OK with task
                        
                    else Task Not Found in DB
                        DB-->>TS: Error (gorm.ErrRecordNotFound)
                        TS-->>CS: Error
                        CS-->>H: Error
                        H-->>C: 404 Not Found
                    end
                end
                
            else Circuit Breaker OPEN/HALF-OPEN
                CB-->>MC: Circuit breaker prevents L2 call
                note over CB: Too many Redis failures, circuit OPEN
                
                alt Non-ErrCacheMiss Error
                    CB-->>MC: err != nil && err != ErrCacheMiss
                    MC->>M: RecordError()
                end
                
                MC->>M: RecordMiss()
                
                %% Skip Redis, go directly to DB
                CS->>TS: GetTaskByID(db, id)
                TS->>DB: SELECT * FROM tasks WHERE id = ?
                
                alt Task Found in DB
                    DB-->>TS: Task record
                    TS-->>CS: Fresh task from DB
                    
                    CS->>MC: Set(cacheKey, task, 30min TTL)
                    MC->>L1: Set(cacheKey, task, 30min TTL)
                    note over MC: L2 skipped due to circuit breaker
                    MC->>M: RecordSet()
                    
                    CS-->>H: Fresh task from DB
                    H-->>C: 200 OK with task
                else Task Not Found in DB
                    DB-->>TS: Error (gorm.ErrRecordNotFound)
                    TS-->>CS: Error
                    CS-->>H: Error
                    H-->>C: 404 Not Found
                end
            end
            
        else No Redis (L2 Cache Unavailable)
            MC->>M: RecordMiss()
            note over MC: Only L1 available (no L2 configured)
            
            %% Direct to Database
            CS->>TS: GetTaskByID(db, id)
            TS->>DB: SELECT * FROM tasks WHERE id = ?
            
            alt Task Found in DB
                DB-->>TS: Task record
                TS-->>CS: Fresh task from DB
                
                CS->>MC: Set(cacheKey, task, 30min TTL)
                MC->>L1: Set(cacheKey, task, 30min TTL)
                MC->>M: RecordSet()
                
                CS-->>H: Fresh task from DB
                H-->>C: 200 OK with task
            else Task Not Found in DB
                DB-->>TS: Error (gorm.ErrRecordNotFound)
                TS-->>CS: Error
                CS-->>H: Error
                H-->>C: 404 Not Found
            end
        end
    end

    %% Cache Invalidation (on Update/Delete)
    note over CS: Cache Invalidation Strategy
    C->>H: PUT /tasks/:id (update task)
    H->>CS: UpdateTask(db, id, updated)
    CS->>TS: UpdateTask(db, id, updated)
    TS->>DB: UPDATE tasks SET ... WHERE id = ?
    
    %% Invalidate specific task cache
    CS->>MC: Delete("task:{id}")
    MC->>L1: Delete("task:{id}")
    MC->>CB: Execute(L2.Delete operation)
    CB->>L2: Delete("task:{id}")
    
    %% Invalidate related caches
    CS->>MC: DeletePattern("user_tasks:{userID}:*")
    CS->>MC: DeletePattern("tasks_paginated:*")
    CS->>MC: Delete("all_tasks")
    
    CS-->>H: Update successful
    H-->>C: 200 OK

    %% Cache Management (on Create)
    note over CS: Create Task with Cache Management
    C->>H: POST /tasks (create task)
    H->>CS: CreateTask(db, task)
    CS->>TS: CreateTask(db, task)
    TS->>DB: INSERT INTO tasks ...
    DB-->>TS: Task created
    TS-->>CS: Created task
    
    %% Cache the new task immediately
    CS->>MC: Set("task:{id}", task, 30min TTL)
    MC->>L1: Set("task:{id}", task, 30min TTL)
    MC->>M: RecordSet()
    
    alt Redis Available
        MC->>CB: Execute(L2.Set operation)
        CB->>L2: Set("task:{id}", task, 30min TTL)
        L2-->>CB: Set successful
        CB-->>MC: Redis cache updated
    else Redis Unavailable
        note over MC: Only cached in L1 memory
    end
    
    %% Invalidate related list caches (FIXED)
    CS->>MC: DeletePattern("user_tasks:{userID}:*")
    CS->>MC: DeletePattern("tasks_paginated:*")
    CS->>MC: Delete("all_tasks")
    note over CS: Ensures new task appears in all listings
    
    CS-->>H: Create successful
    H-->>C: 201 Created with task

    %% Cache Invalidation (on Delete)
    note over CS: Delete Task with Cache Cleanup
    C->>H: DELETE /tasks/:id
    H->>CS: DeleteTask(db, id)
    CS->>TS: GetTaskByID(db, id) for userID
    TS->>DB: SELECT * FROM tasks WHERE id = ?
    DB-->>TS: Task details (for cleanup)
    TS-->>CS: Task info
    
    CS->>TS: DeleteTask(db, id)
    TS->>DB: DELETE FROM tasks WHERE id = ?
    DB-->>TS: Task deleted
    TS-->>CS: Delete successful
    
    %% Complete cache cleanup
    CS->>MC: Delete("task:{id}")
    MC->>L1: Delete("task:{id}")
    MC->>CB: Execute(L2.Delete operation)
    CB->>L2: Delete("task:{id}")
    
    CS->>MC: DeletePattern("user_tasks:{userID}:*")
    CS->>MC: DeletePattern("tasks_paginated:*")
    CS->>MC: Delete("all_tasks")
    
    CS-->>H: Delete successful
    H-->>C: 204 No Content
```

### Authentication & Authorization Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant AH as AuthHandler
    participant AS as AuthService
    participant DB as Database
    participant M as AuthzMiddleware
    participant AUTHZ as AuthorizationService
    participant TH as TaskHandler

    %% Login Flow
    C->>AH: POST /auth/login {email, password}
    AH->>AH: Validate request & normalize email
    AH->>AS: LoginUser(email, password)
    AS->>DB: SELECT user WHERE email=? AND is_active=true
    DB-->>AS: User found
    AS->>AS: VerifyPassword(hashedPassword, plainPassword)
    
    alt User is inactive
        AH-->>C: 403 Forbidden (account_disabled)
    else Login successful
        AH->>AS: GenerateToken(userID)
        AS->>DB: SELECT role FROM user_roles WHERE user_id=? LIMIT 1
        AS->>DB: SELECT DISTINCT permissions FROM role_permissions JOIN user_roles
        AS->>AS: Create JWT with claims (user_id, role, permissions, iss, aud, iat, exp)
        AS->>DB: INSERT refresh_token (UUID) with 7-day expiry
        AS-->>AH: JWT (1hr expiry) + Refresh Token UUID
        
        AH->>DB: UPDATE user SET last_login_at=NOW()
        AH->>AS: GetUserPermissions(userID)
        AS->>DB: SELECT DISTINCT permissions via role_permissions
        AS-->>AH: User permissions array
        
        AH-->>C: 200 OK {access_token, refresh_token, token_type: "Bearer", expires_in: 3600, user_profile, permissions}
    end

    %% Authenticated Request Flow
    C->>M: GET /tasks (with Authorization: Bearer JWT)
    M->>M: Extract Bearer token from Authorization header
    
    alt Missing or invalid token format
        M-->>C: 401 Unauthorized (missing_token/invalid_token_format)
    else Token validation
        M->>M: jwt.Parse(tokenStr) with JWT_SECRET
        M->>M: Validate HMAC signature
        M->>M: Validate token.Valid flag
        M->>M: Check exp claim (expiry time)
        M->>M: Check iss=taskify-backend (issuer)
        
        alt Token expired, invalid signature, or wrong issuer
            M-->>C: 401 Unauthorized (expired_token/invalid_token/invalid_issuer)
        else Token valid
            M->>M: Extract claims (user_id, role, permissions)
            M->>M: Check role requirements (if configured)
            M->>M: Check permission requirements (if configured)
            
            alt Insufficient role or permissions
                M-->>C: 403 Forbidden (insufficient_role/missing_permission)
            else Authorization granted
                M->>M: Set context: user_id, user_role, user_permissions
                M->>TH: Request continues with user context
                
                %% Optional: Advanced Authorization Check
                note over TH,AUTHZ: Advanced RBAC/ABAC checks (if needed for resource ownership)
                TH->>AUTHZ: IsAuthorized(AuthorizationRequest)
                AUTHZ->>DB: HasPermission() - check role_permissions
                AUTHZ->>DB: Query user_attributes for ABAC
                AUTHZ->>AUTHZ: Evaluate RBAC (permissions) & ABAC (ownership/attributes) policies
                AUTHZ-->>TH: AuthorizationDecision {decision, reason, policy_type}
                
                alt Access denied
                    TH-->>C: 403 Forbidden {error, reason}
                else Access granted
                    TH->>TH: ProcessRequest() with authorized context
                    TH->>DB: Execute business logic with user context
                    TH-->>C: 200 OK with requested data
                end
            end
        end
    end

    %% Token Refresh Flow
    C->>AH: POST /auth/refresh {refresh_token}
    AH->>AS: RefreshToken(refreshToken)
    AS->>DB: SELECT token WHERE refresh_token=? AND expires_at > NOW()
    
    alt Invalid or expired refresh token
        AS-->>AH: Error (token not found/expired)
        AH-->>C: 401 Unauthorized (invalid_token)
    else Valid refresh token
        AS->>AS: GenerateToken(userID) - create new JWT & refresh token
        AS->>DB: DELETE old refresh_token
        AS->>DB: INSERT new refresh_token with 7-day expiry
        AS-->>AH: New JWT + New Refresh Token
        AH-->>C: 200 OK {access_token, refresh_token, token_type: "Bearer", expires_in: 3600}
    end

    %% Logout Flow
    C->>AH: POST /auth/logout {refresh_token}
    AH->>AS: RevokeToken(refreshToken)
    AS->>DB: DELETE FROM tokens WHERE refresh_token=?
    AS-->>AH: Token deleted (always returns success)
    AH-->>C: 200 OK {message: "Successfully logged out"}
```

### Job/Worker Sequence Flow

```mermaid
sequenceDiagram
    participant C as Client/Application
    participant UCM as UnifiedCacheManager
    participant ICW as IntegratedCacheWarmer
    participant JR as JobRouter
    participant DCW as DistributedCacheWorker
    participant WP as WorkerPool
    participant W as Worker Goroutine
    participant RC as ResultCollector
    participant JS as JobScheduler
    participant PQ as PriorityQueue
    participant Redis as Redis Queue
    participant Cache as Cache

    %% Initialization Phase
    C->>UCM: NewUnifiedCacheManager(cache, config)
    UCM->>UCM: determineMode(config)
    note over UCM: Pings Redis to test availability
    
    alt Integrated Mode (Redis Available)
        UCM->>ICW: NewIntegratedCacheWarmer(cache, redisClient, strategy)
        ICW->>ICW: Create JobRouter inline (redisAvailable, localCapacity, distributedQueue)
        ICW->>WP: NewWorkerPool(workers, cache)
        ICW->>ICW: Create legacy CacheWarmer for scheduler
        ICW->>JS: NewJobScheduler(legacyWarmer, workerPool)
        ICW->>PQ: NewPriorityQueue()
        
        alt Redis Client Provided
            ICW->>Redis: Ping(ctx) with 2s timeout
            alt Redis Ping Success
                ICW->>JR: Set redisAvailable = true
                note over ICW: Redis available for distributed processing
            else Redis Ping Failed
                ICW->>JR: Set redisAvailable = false
                note over ICW: Falls back to local-only mode
            end
        else No Redis Client
            ICW->>JR: Set redisAvailable = false
        end
        
        note over ICW: Distributed workers NOT created yet (created in Start)
    else Legacy Mode (Redis Unavailable)
        UCM->>CW: NewCacheWarmer(cache, strategy)
        note over UCM: Falls back to legacy in-memory system
    end

    %% Startup Phase
    C->>UCM: Start()
    UCM->>ICW: Start()
    
    ICW->>WP: Start()
    loop Worker Initialization
        WP->>W: Start worker(id) goroutine
        W->>W: Listen on jobCh channel
        note over W: Worker waits for jobs
    end
    
    WP->>RC: Start resultCollector() goroutine
    note over RC: Collects job results and updates metrics
    
    alt Redis Available (jobRouter.redisAvailable)
        loop Create Distributed Workers (strategy.ConcurrentJobs)
            ICW->>DCW: Create DistributedCacheWorker{id, warmer, redisClient, queueName}
            ICW->>DCW: start() - launch goroutine
            DCW->>DCW: Start polling loop with 100ms ticker
            note over DCW: Polls Redis queues every 100ms
        end
        note over ICW: Started N distributed cache workers
    else Redis Unavailable
        note over ICW: Skip distributed worker creation (local-only mode)
    end
    
    alt Scheduler Enabled
        ICW->>JS: Start()
        JS->>JS: Start schedulerLoop()
    end

    %% Job Enqueue Phase
    C->>UCM: EnqueueWarmupJob(key, data, ttl, priority)
    UCM->>ICW: EnqueueWarmupJob(key, data, ttl, priority)
    ICW->>ICW: Create DistributedCacheJob{ID, Type, Payload, Priority, MaxTries}
    ICW->>ICW: routeJob(job) - Check routing conditions
    
    alt Case 1: Redis Unavailable (!redisAvailable)
        note over ICW: Redis not available, route to local
        ICW->>ICW: localJobs++
        ICW->>ICW: enqueueToLocal(job)
        ICW->>ICW: Convert DistributedCacheJob to WarmupJob
        ICW->>WP: SubmitJob(warmupJob)
        WP->>W: Send job via jobCh
        W->>W: processJob(job)
        W->>Cache: Set(key, data, TTL)
        Cache-->>W: Success/Error
        W->>RC: Send JobResult via resultCh
        RC->>WP: Update metrics (jobsProcessed, errors)
        
    else Case 2: Scheduled Job (Type == CacheJobScheduled)
        note over ICW: Scheduled jobs always go to distributed
        ICW->>ICW: distributedJobs++
        ICW->>ICW: enqueueToRedis(job)
        ICW->>Redis: ZAdd(cache_jobs, prioritizedJob)
        note over ICW: Score = now.Unix() - priority
        Redis-->>ICW: Job queued
        
    else Case 3: High Priority Job (priority == 1)
        note over ICW: High priority goes to local for fast processing
        ICW->>ICW: localJobs++
        ICW->>ICW: enqueueToLocal(job)
        ICW->>ICW: Convert DistributedCacheJob to WarmupJob
        ICW->>WP: SubmitJob(warmupJob)
        WP->>W: Send job via jobCh
        W->>W: processJob(job)
        W->>Cache: Set(key, data, TTL)
        Cache-->>W: Success/Error
        W->>RC: Send JobResult via resultCh
        RC->>WP: Update metrics (jobsProcessed, errors)
        
    else Case 4: Local Queue Full (len(jobCh) > ConcurrentJobs)
        note over ICW: Local queue full, overflow to distributed
        ICW->>ICW: distributedJobs++
        ICW->>ICW: enqueueToRedis(job)
        ICW->>Redis: ZAdd(cache_jobs, prioritizedJob)
        Redis-->>ICW: Job queued
        
    else Case 5: Default (Normal Priority, Queue Available)
        note over ICW: Normal priority with space, prefer local
        ICW->>ICW: localJobs++
        ICW->>ICW: enqueueToLocal(job)
        ICW->>ICW: Convert DistributedCacheJob to WarmupJob
        ICW->>WP: SubmitJob(warmupJob)
        WP->>W: Send job via jobCh
        W->>W: processJob(job)
        W->>Cache: Set(key, data, TTL)
        Cache-->>W: Success/Error
        W->>RC: Send JobResult via resultCh
        RC->>WP: Update metrics (jobsProcessed, errors)
    end
    
    %% Distributed Processing Loop (for jobs routed to Redis)
    loop Distributed Worker Polling (every 100ms)
        DCW->>DCW: processJobs() - Called by ticker
        DCW->>DCW: moveDelayedJobs(ctx)
        DCW->>Redis: ZRangeByScore(cache_jobs_delayed, "-inf", now, Count:10)
        Redis-->>DCW: Ready delayed jobs (up to 10)
        
        loop For each ready delayed job
            DCW->>Redis: ZRem(cache_jobs_delayed, jobData)
            note over DCW: Calculate score = now.Unix() - priority
            DCW->>Redis: ZAdd(cache_jobs, jobData, score)
        end
        
        DCW->>DCW: processImmediateJobs(ctx)
        DCW->>Redis: ZPopMin(cache_jobs) - Get highest priority job
        Redis-->>DCW: Next priority job (lowest score = highest priority)
        
        alt Job Retrieved
            DCW->>DCW: executeJob(job)
            note over DCW: Increment job.Attempts
            
            alt Job Type: Warmup or Scheduled (CacheJobWarmup, CacheJobScheduled)
                DCW->>DCW: handleWarmupJob(job)
                DCW->>Cache: Set(key, data, TTL)
            else Job Type: Batch (CacheJobBatch)
                DCW->>DCW: handleBatchJob(job)
                loop For each key in batch
                    DCW->>Cache: Set(key, data, TTL)
                end
            else Job Type: Validation (CacheJobValidation)
                DCW->>DCW: handleValidationJob(job)
                DCW->>Cache: Exists(key)
                alt Validation Fails
                    DCW->>ICW: Re-enqueue warmup job
                end
            else Job Type: Eviction (CacheJobEviction)
                DCW->>DCW: handleEvictionJob(job)
                DCW->>Cache: Delete(key)
            end
            
            Cache-->>DCW: Operation result
            
            alt Job Success
                DCW->>ICW: Increment processedJobs
                note over DCW: Log: job completed successfully
            else Job Failure
                DCW->>ICW: Increment failedJobs
                note over DCW: Log: job failed with error
                
                alt Retries Available (attempts < maxTries)
                    DCW->>DCW: Calculate retry delay (attempts * 1 second)
                    DCW->>DCW: Set job.ProcessAt = now + retryDelay
                    DCW->>Redis: ZAdd(cache_jobs_delayed, retryJob)
                    note over DCW: Score = ProcessAt.Unix()
                    DCW->>ICW: Increment retryJobs
                    note over DCW: Linear backoff: delay = attempts * 1 second
                else Max Retries Exceeded
                    DCW->>Redis: LPush(cache_jobs_dlq, failedJob)
                    note over DCW: Move to dead letter queue (DLQ)
                end
            end
        end
    end

    %% Scheduled Job Processing
    C->>UCM: EnqueueScheduledWarmup(key, data, ttl, processAt, priority)
    UCM->>ICW: EnqueueScheduledWarmup(...)
    ICW->>ICW: Create DistributedCacheJob{Type: CacheJobScheduled}
    
    alt Redis Available (redisAvailable)
        ICW->>ICW: Check if processAt is in future
        
        alt Future Job (processAt > now)
            ICW->>Redis: ZAdd(cache_jobs_delayed, scheduledJob)
            note over ICW: Score = processAt.Unix()
        else Immediate Job (processAt <= now)
            ICW->>Redis: ZAdd(cache_jobs, immediateJob)
            note over ICW: Score = now.Unix() - priority
        end
    else Redis Unavailable
        note over ICW: Log warning: "Redis unavailable for scheduled job, executing immediately"
        ICW->>ICW: enqueueToLocal(job)
        ICW->>WP: SubmitJob(warmupJob)
        WP->>W: Process immediately via jobCh
    end

    %% Batch Job Processing
    C->>UCM: EnqueueBatchWarmupJob(keys, data, priority)
    UCM->>ICW: EnqueueBatchWarmupJob(...)
    ICW->>ICW: Create DistributedCacheJob{Type: CacheJobBatch}
    ICW->>ICW: routeJob(batchJob) - Apply routing logic
    
    alt Routed to Local (priority==1 OR queue available OR Redis unavailable)
        ICW->>ICW: enqueueToLocal(job)
        note over ICW: Convert batch job to multiple WarmupJobs
        
        loop For each key in batch
            ICW->>ICW: Create WarmupJob{key, data, ttl, priority}
            ICW->>WP: SubmitJob(warmupJob)
        end
        
        par Concurrent Batch Processing
            WP->>W: Process job for key1 via jobCh
            W->>Cache: Set(key1, data, TTL)
        and
            WP->>W: Process job for key2 via jobCh
            W->>Cache: Set(key2, data, TTL)
        and
            WP->>W: Process job for key3 via jobCh
            W->>Cache: Set(key3, data, TTL)
        end
        
        loop Collect Results
            W->>RC: Send JobResult via resultCh
            RC->>WP: Update batch metrics
        end
        
    else Routed to Distributed (queue full OR normal priority)
        ICW->>ICW: enqueueToRedis(job)
        ICW->>Redis: ZAdd(cache_jobs, batchJob)
        note over ICW: Single batch job queued for distributed processing
        
        DCW->>Redis: ZPopMin(cache_jobs)
        DCW->>DCW: executeJob(batchJob)
        DCW->>DCW: handleBatchJob(job)
        
        loop For each key sequentially
            DCW->>Cache: Set(key, data, TTL)
        end
    end

    %% Metrics & Monitoring
    C->>UCM: GetMetrics()
    UCM->>ICW: GetMetrics()
    
    par Collect Metrics
        ICW->>WP: GetStats()
        WP-->>ICW: Local worker stats (jobsProcessed, errors, totalDuration)
    and
        ICW->>JS: GetStats()
        JS-->>ICW: Scheduler stats
    and
        ICW->>PQ: Len()
        PQ-->>ICW: Queue sizes
    and
        ICW->>Redis: ZCard operations on all queues
        Redis-->>ICW: Distributed queue metrics (immediate, delayed, dlq)
    end
    
    ICW-->>UCM: Comprehensive metrics
    UCM-->>C: Unified metrics response

    %% Health Check
    C->>UCM: HealthCheck()
    UCM->>ICW: HealthCheck()
    
    par Health Checks
        ICW->>Redis: Ping() with 2s timeout
        Redis-->>ICW: Connection status
    and
        ICW->>WP: IsRunning()
        WP-->>ICW: Worker pool status
    and
        ICW->>JS: IsRunning()
        JS-->>ICW: Scheduler status
    end
    
    ICW-->>UCM: Health status with Redis connectivity
    UCM-->>C: Overall system health```
    UCM-->>C: System health report

    %% Graceful Shutdown
    C->>UCM: Stop()
    UCM->>ICW: Stop()
    
    ICW->>JS: Stop()
    JS->>JS: Cancel context & stop loops
    
    loop For each distributed worker
        ICW->>DCW: Close stopCh
        DCW->>DCW: Complete current job & exit
    end
    
    ICW->>WP: Stop()
    WP->>WP: Close jobCh
    loop For each worker
        WP->>W: Workers receive shutdown signal
        W->>W: Complete current job & exit
    end
    WP->>WP: Wait for all workers (wg.Wait())
    
    ICW->>ICW: Close stopCh & cleanup
    
    note over UCM,Cache: All components gracefully shut down
```
## Deployment Architecture

### Development Configuration (`run-dev.sh`)

**Production-like development environment using scalable architecture - for local development with nginx load balancer.**

This configuration uses `docker-compose.scalable.yml` but starts only core services (postgres, redis, backend, frontend, nginx), excluding monitoring (Prometheus/Grafana) for faster startup.

```mermaid
graph TB
    subgraph "Development Environment (run-dev.sh)"
        subgraph "Client Access"
            INTERNET[Local Developers<br/>http://localhost]
        end
        
        subgraph "Load Balancer & Proxy"
            NGINX[nginx:alpine<br/>Container: task-manager-nginx<br/>Ports: 80, 443<br/>Rate Limiting Enabled]
        end
        
        subgraph "Application Layer"
            BACKEND[Backend API<br/>Go + Gin Framework<br/>Port: 8080 internal<br/>No container_name<br/>Scalable Ready]
            FRONTEND[Frontend App<br/>React + nginx<br/>Container: task-manager-frontend<br/>Port: 3000:80]
        end
        
        subgraph "Data & Cache Layer"
            POSTGRES[(PostgreSQL 15 Alpine<br/>Container: task-manager-postgres<br/>Port: 5432:5432<br/>Health Checks Active<br/>Connection Pool: 25)]
            REDIS[(Redis 7 Alpine<br/>Container: task-manager-redis<br/>Port: 6379:6379<br/>LRU + AOF Persistence<br/>Pool: 10 conns)]
        end
        
        subgraph "Infrastructure"
            VOLUMES[Docker Volumes<br/>postgres_data<br/>redis_data]
            NETWORK[Named Network<br/>task-manager-network<br/>Bridge Driver]
        end
    end

    %% External Access Flow
    INTERNET -->|"Port 80<br/>HTTP"| NGINX
    INTERNET -->|"Port 443<br/>HTTPS (ready)"| NGINX
    
    %% Load Balancer Routes
    NGINX -->|"/api/v1/*<br/>Backend API"| BACKEND
    NGINX -->|"/<br/>Static Assets"| FRONTEND
    NGINX -->|"/health<br/>Health Check"| BACKEND
    NGINX -->|"/metrics<br/>Prometheus Metrics"| BACKEND
    NGINX -->|"/nginx-health<br/>LB Health"| NGINX
    
    %% Application Dependencies
    BACKEND -->|"SQL Queries<br/>Connection Pool<br/>Max 25 conns"| POSTGRES
    BACKEND -->|"Cache + Queue<br/>Redis Pool<br/>10 connections"| REDIS
    FRONTEND -->|"API Calls via<br/>nginx proxy"| NGINX
    
    %% Infrastructure Dependencies
    POSTGRES -.->|"Persists<br/>Database Data"| VOLUMES
    REDIS -.->|"Persists<br/>Cache + AOF"| VOLUMES
    
    BACKEND -.->|"Isolated<br/>Communication"| NETWORK
    FRONTEND -.->|"Isolated<br/>Communication"| NETWORK
    POSTGRES -.->|"Isolated<br/>Communication"| NETWORK
    REDIS -.->|"Isolated<br/>Communication"| NETWORK
    NGINX -.->|"Isolated<br/>Communication"| NETWORK

    %% Styling
    classDef appClass fill:#e8f5e8,stroke:#4caf50,stroke-width:2px
    classDef dataClass fill:#fce4ec,stroke:#e91e63,stroke-width:2px
    classDef infraClass fill:#e3f2fd,stroke:#2196f3,stroke-width:2px
    classDef lbClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    classDef clientClass fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px

    class BACKEND appClass
    class FRONTEND appClass
    class POSTGRES dataClass
    class REDIS dataClass
    class VOLUMES infraClass
    class NETWORK infraClass
    class NGINX lbClass
    class INTERNET clientClass
```

---

### Scalable Production Configuration

- This configuration is designed for production workloads with dynamic scaling capabilities, load balancing, and comprehensive monitoring. The backend can be scaled to multiple instances without code changes or configuration updates.  

```mermaid
graph TB
    subgraph "Scalable Deployment (Manual Scaling)"
        subgraph "External Access"
            USERS[Users/Clients]
        end
        
        subgraph "Load Balancer"
            LB[nginx Load Balancer<br/>Ports: 80, 443<br/>Rate Limiting & Security]
        end
        
        subgraph "Application Tier (Scalable)"
            APP1[Backend Instance 1<br/>./scale.sh N]
            APP_N[Backend Instance N<br/>Auto-scaled via<br/>Docker Compose]
            APP1 -.->|"Can scale to"| APP_N
        end
        
        subgraph "Frontend"
            WEB[Frontend Application<br/>React/nginx<br/>Static Assets]
        end
        
        subgraph "Data Layer (Single Instance)"
            DB[(PostgreSQL 15<br/>Single Instance<br/>Health Checks<br/>Connection Pooling)]
            CACHE[(Redis 7<br/>Single Instance<br/>LRU Eviction<br/>Persistence)]
        end
        
        subgraph "Monitoring & Observability"
            PROM[Prometheus<br/>Metrics Scraping<br/>15s Intervals]
            GRAF[Grafana<br/>Dashboards<br/>admin/admin]
        end
        
        subgraph "Configuration"
            ENV[Environment Variables<br/>.env Configuration<br/>Port Management]
            SCRIPTS[Management Scripts<br/>./scale.sh<br/>./run-prod.sh<br/>./setup-scalable.sh]
        end
    end

    %% User Flow
    USERS --> LB
    
    %% Load Balancing
    LB -->|"least_conn"| APP1
    LB -.->|"distributes to"| APP_N
    LB --> WEB
    
    %% Data Connections
    APP1 --> DB
    APP_N --> DB
    APP1 --> CACHE
    APP_N --> CACHE
    
    %% Monitoring
    PROM --> APP1
    PROM -.-> APP_N
    PROM --> DB
    PROM --> CACHE
    GRAF --> PROM
    
    %% Configuration
    ENV -.-> APP1
    ENV -.-> APP_N
    SCRIPTS -.-> LB

    %% Styling
    classDef scaleClass fill:#e8f5e8,stroke:#4caf50,stroke-width:3px
    classDef singleClass fill:#fce4ec,stroke:#e91e63,stroke-width:2px
    classDef monitorClass fill:#f1f8e9,stroke:#8bc34a,stroke-width:2px
    classDef configClass fill:#e3f2fd,stroke:#2196f3,stroke-width:2px
    classDef lbClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    classDef clientClass fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px

    class APP1 scaleClass
    class APP_N scaleClass
    class DB singleClass
    class CACHE singleClass
    class PROM monitorClass
    class GRAF monitorClass
    class ENV configClass
    class SCRIPTS configClass
    class LB lbClass
    class WEB scaleClass
    class USERS clientClass
```

**Scaling Behavior:**
- **Automatic Service Discovery**: New instances automatically join the nginx upstream pool
- **Session-Independent**: JWT-based stateless authentication supports any backend instance
- **Shared Data Layer**: All backends connect to single PostgreSQL and Redis instances
- **Cache Coherence**: Redis L2 cache synchronized across all backend instances

**Performance Characteristics:**
| Metric | Single Instance | 3 Instances | 5 Instances |
|--------|----------------|-------------|-------------|
| **Max Throughput** | ~500 req/s | ~1,500 req/s | ~2,500 req/s |
| **Concurrent Users** | ~100 | ~300 | ~500 |
| **Failover Time** | N/A (single point of failure) | <1 second | <1 second |
| **CPU Utilization** | High | Balanced | Optimal |

**Use Cases:**
- Production environments with >100 concurrent users
- Applications requiring high availability (99.9% uptime SLA)
- Traffic spike handling (e.g., Black Friday, product launches)
- Blue-green and canary deployments
- Kubernetes-ready architecture (easy migration path)


---
