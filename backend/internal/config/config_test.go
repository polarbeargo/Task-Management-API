package config

import (
	"os"
	"testing"
	"time"
)

func setEnvVars(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func clearEnvVars(vars []string) {
	for _, k := range vars {
		os.Unsetenv(k)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	envVars := []string{
		"HOST", "PORT", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT", "ENVIRONMENT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_POOL_SIZE",
		"REDIS_MIN_IDLE_CONNS", "REDIS_MAX_RETRIES", "REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT",
		"WORKER_CONCURRENCY", "WORKER_POLL_INTERVAL",
		"JWT_SECRET", "ACCESS_TOKEN_TTL", "REFRESH_TOKEN_TTL", "BCRYPT_COST",
		"RATE_LIMIT_ENABLED", "RATE_LIMIT_RPM", "RATE_LIMIT_BURST", "RATE_LIMIT_CLEANUP",
	}
	clearEnvVars(envVars)

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error with default config, got: %v", err)
	}

	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got %s", config.Server.Host)
	}

	if config.Server.Port != "8080" {
		t.Errorf("Expected default port '8080', got %s", config.Server.Port)
	}

	if config.Server.Environment != "development" {
		t.Errorf("Expected default environment 'development', got %s", config.Server.Environment)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected default DB host 'localhost', got %s", config.Database.Host)
	}

	if config.Database.Port != "5432" {
		t.Errorf("Expected default DB port '5432', got %s", config.Database.Port)
	}

	if config.Database.User != "postgres" {
		t.Errorf("Expected default DB user 'postgres', got %s", config.Database.User)
	}

	if config.Database.Name != "task_manager" {
		t.Errorf("Expected default DB name 'task_manager', got %s", config.Database.Name)
	}

	if config.Database.MaxOpenConns != 25 {
		t.Errorf("Expected default max open conns 25, got %d", config.Database.MaxOpenConns)
	}

	if config.Redis.Host != "localhost" {
		t.Errorf("Expected default Redis host 'localhost', got %s", config.Redis.Host)
	}

	if config.Redis.Port != "6379" {
		t.Errorf("Expected default Redis port '6379', got %s", config.Redis.Port)
	}

	if config.Redis.DB != 0 {
		t.Errorf("Expected default Redis DB 0, got %d", config.Redis.DB)
	}

	if config.Redis.PoolSize != 10 {
		t.Errorf("Expected default Redis pool size 10, got %d", config.Redis.PoolSize)
	}

	if config.Worker.Concurrency != 4 {
		t.Errorf("Expected default worker concurrency 4, got %d", config.Worker.Concurrency)
	}

	if len(config.Worker.Queues) != 3 {
		t.Errorf("Expected 3 default queues, got %d", len(config.Worker.Queues))
	}

	if config.Auth.BCryptCost != 10 {
		t.Errorf("Expected default bcrypt cost 10, got %d", config.Auth.BCryptCost)
	}

	if !config.RateLimit.Enabled {
		t.Error("Expected rate limiting to be enabled by default")
	}

	if config.RateLimit.RequestsPerMin != 100 {
		t.Errorf("Expected default requests per minute 100, got %d", config.RateLimit.RequestsPerMin)
	}
}

func TestLoadConfig_CustomEnvironment(t *testing.T) {
	envVars := map[string]string{
		"HOST":               "0.0.0.0",
		"PORT":               "9000",
		"ENVIRONMENT":        "production",
		"DB_HOST":            "db.example.com",
		"DB_PORT":            "5433",
		"DB_USER":            "app_user",
		"DB_PASSWORD":        "secure_password",
		"DB_NAME":            "production_db",
		"DB_MAX_OPEN_CONNS":  "50",
		"REDIS_HOST":         "redis.example.com",
		"REDIS_PORT":         "6380",
		"REDIS_PASSWORD":     "redis_pass",
		"REDIS_DB":           "1",
		"WORKER_CONCURRENCY": "8",
		"JWT_SECRET":         "super-secret-key",
		"RATE_LIMIT_ENABLED": "false",
		"RATE_LIMIT_RPM":     "200",
		"READ_TIMEOUT":       "45s",
		"WRITE_TIMEOUT":      "45s",
		"ACCESS_TOKEN_TTL":   "30m",
		"REFRESH_TOKEN_TTL":  "720h",
	}

	setEnvVars(envVars)
	defer func() {
		var keys []string
		for k := range envVars {
			keys = append(keys, k)
		}
		clearEnvVars(keys)
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error with custom config, got: %v", err)
	}

	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got %s", config.Server.Host)
	}

	if config.Server.Port != "9000" {
		t.Errorf("Expected port '9000', got %s", config.Server.Port)
	}

	if config.Server.Environment != "production" {
		t.Errorf("Expected environment 'production', got %s", config.Server.Environment)
	}

	if config.Database.Host != "db.example.com" {
		t.Errorf("Expected DB host 'db.example.com', got %s", config.Database.Host)
	}

	if config.Database.Password != "secure_password" {
		t.Errorf("Expected DB password 'secure_password', got %s", config.Database.Password)
	}

	if config.Database.MaxOpenConns != 50 {
		t.Errorf("Expected max open conns 50, got %d", config.Database.MaxOpenConns)
	}

	if config.Redis.Host != "redis.example.com" {
		t.Errorf("Expected Redis host 'redis.example.com', got %s", config.Redis.Host)
	}

	if config.Redis.DB != 1 {
		t.Errorf("Expected Redis DB 1, got %d", config.Redis.DB)
	}

	if config.Worker.Concurrency != 8 {
		t.Errorf("Expected worker concurrency 8, got %d", config.Worker.Concurrency)
	}

	if config.Auth.JWTSecret != "super-secret-key" {
		t.Errorf("Expected JWT secret 'super-secret-key', got %s", config.Auth.JWTSecret)
	}

	if config.RateLimit.Enabled {
		t.Error("Expected rate limiting to be disabled")
	}

	if config.RateLimit.RequestsPerMin != 200 {
		t.Errorf("Expected requests per minute 200, got %d", config.RateLimit.RequestsPerMin)
	}

	if config.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Expected read timeout 45s, got %v", config.Server.ReadTimeout)
	}

	if config.Auth.AccessTokenTTL != 30*time.Minute {
		t.Errorf("Expected access token TTL 30m, got %v", config.Auth.AccessTokenTTL)
	}

	if config.Auth.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("Expected refresh token TTL 720h, got %v", config.Auth.RefreshTokenTTL)
	}
}

func TestLoadConfig_ProductionValidation(t *testing.T) {
	envVars := map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  "secure-jwt-secret",
	}

	setEnvVars(envVars)
	defer func() {
		var keys []string
		for k := range envVars {
			keys = append(keys, k)
		}
		clearEnvVars(keys)
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing database password in production")
	}

	if err.Error() != "database password is required in production" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadConfig_ProductionJWTValidation(t *testing.T) {
	envVars := map[string]string{
		"ENVIRONMENT": "production",
		"DB_PASSWORD": "secure-db-password",
	}

	setEnvVars(envVars)
	defer func() {
		var keys []string
		for k := range envVars {
			keys = append(keys, k)
		}
		clearEnvVars(keys)
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for default JWT secret in production")
	}

	if err.Error() != "JWT secret must be set in production" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestConfig_GetDatabaseDSN(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
			SSLMode:  "require",
		},
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require"
	actual := config.GetDatabaseDSN()

	if actual != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, actual)
	}
}

func TestConfig_GetRedisAddr(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			Host: "redis.example.com",
			Port: "6380",
		},
	}

	expected := "redis.example.com:6380"
	actual := config.GetRedisAddr()

	if actual != expected {
		t.Errorf("Expected Redis addr '%s', got '%s'", expected, actual)
	}
}

func TestConfig_GetServerAddr(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "9000",
		},
	}

	expected := "0.0.0.0:9000"
	actual := config.GetServerAddr()

	if actual != expected {
		t.Errorf("Expected server addr '%s', got '%s'", expected, actual)
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		environment string
		expected    bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
		{"test", false},
		{"", false},
	}

	for _, test := range tests {
		config := &Config{
			Server: ServerConfig{
				Environment: test.environment,
			},
		}

		actual := config.IsProduction()
		if actual != test.expected {
			t.Errorf("For environment '%s', expected IsProduction() = %v, got %v",
				test.environment, test.expected, actual)
		}
	}
}

func TestGetEnv(t *testing.T) {
	key := "TEST_ENV_VAR"
	defaultValue := "default"

	os.Unsetenv(key)
	result := getEnv(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value '%s', got '%s'", defaultValue, result)
	}

	expectedValue := "custom_value"
	os.Setenv(key, expectedValue)
	defer os.Unsetenv(key)

	result = getEnv(key, defaultValue)
	if result != expectedValue {
		t.Errorf("Expected env value '%s', got '%s'", expectedValue, result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	key := "TEST_INT_VAR"
	defaultValue := 42

	os.Unsetenv(key)
	result := getEnvAsInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value %d, got %d", defaultValue, result)
	}

	os.Setenv(key, "100")
	defer os.Unsetenv(key)

	result = getEnvAsInt(key, defaultValue)
	if result != 100 {
		t.Errorf("Expected env value 100, got %d", result)
	}

	os.Setenv(key, "not-a-number")
	result = getEnvAsInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value %d for invalid int, got %d", defaultValue, result)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	key := "TEST_BOOL_VAR"
	defaultValue := true

	os.Unsetenv(key)
	result := getEnvAsBool(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value %v, got %v", defaultValue, result)
	}

	testCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"True", true},
		{"False", false},
		{"invalid", defaultValue}, 
	}

	for _, tc := range testCases {
		os.Setenv(key, tc.value)
		result = getEnvAsBool(key, defaultValue)
		if result != tc.expected {
			t.Errorf("For value '%s', expected %v, got %v", tc.value, tc.expected, result)
		}
	}

	os.Unsetenv(key)
}

func TestGetEnvAsDuration(t *testing.T) {
	key := "TEST_DURATION_VAR"
	defaultValue := 30 * time.Second

	os.Unsetenv(key)
	result := getEnvAsDuration(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value %v, got %v", defaultValue, result)
	}

	os.Setenv(key, "5m")
	defer os.Unsetenv(key)

	result = getEnvAsDuration(key, defaultValue)
	if result != 5*time.Minute {
		t.Errorf("Expected env value 5m, got %v", result)
	}

	os.Setenv(key, "not-a-duration")
	result = getEnvAsDuration(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default value %v for invalid duration, got %v", defaultValue, result)
	}
}

func TestConfigValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		hasError bool
		errorMsg string
	}{
		{
			name: "Production with all required fields",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
				"DB_PASSWORD": "secure-password",
				"JWT_SECRET":  "secure-jwt-secret",
			},
			hasError: false,
		},
		{
			name: "Development with default JWT secret",
			envVars: map[string]string{
				"ENVIRONMENT": "development",
			},
			hasError: false,
		},
		{
			name: "Staging environment (not production)",
			envVars: map[string]string{
				"ENVIRONMENT": "staging",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := []string{"ENVIRONMENT", "DB_PASSWORD", "JWT_SECRET"}
			clearEnvVars(envVars)

			setEnvVars(tt.envVars)
			defer func() {
				var keys []string
				for k := range tt.envVars {
					keys = append(keys, k)
				}
				clearEnvVars(keys)
			}()

			config, err := LoadConfig()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if config == nil {
					t.Error("Expected config to be loaded")
				}
			}
		})
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	envVars := map[string]string{
		"HOST":        "0.0.0.0",
		"PORT":        "8080",
		"ENVIRONMENT": "production",
		"DB_PASSWORD": "password",
		"JWT_SECRET":  "secret",
	}
	setEnvVars(envVars)
	defer func() {
		var keys []string
		for k := range envVars {
			keys = append(keys, k)
		}
		clearEnvVars(keys)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

func BenchmarkGetDatabaseDSN(b *testing.B) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "password",
			Name:     "database",
			SSLMode:  "disable",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetDatabaseDSN()
	}
}

func BenchmarkGetEnvAsInt(b *testing.B) {
	os.Setenv("BENCH_INT", "42")
	defer os.Unsetenv("BENCH_INT")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getEnvAsInt("BENCH_INT", 0)
	}
}

func BenchmarkGetEnvAsDuration(b *testing.B) {
	os.Setenv("BENCH_DURATION", "30s")
	defer os.Unsetenv("BENCH_DURATION")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getEnvAsDuration("BENCH_DURATION", time.Second)
	}
}
