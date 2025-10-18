package database

import (
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns to be 25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns to be 10, got %d", config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime to be 1 hour, got %v", config.ConnMaxLifetime)
	}

	if config.ConnMaxIdleTime != time.Minute*30 {
		t.Errorf("Expected ConnMaxIdleTime to be 30 minutes, got %v", config.ConnMaxIdleTime)
	}

	if config.LogLevel != logger.Info {
		t.Errorf("Expected LogLevel to be Info, got %v", config.LogLevel)
	}
}

func TestNewDatabasePool_WithNilConfig(t *testing.T) {
	_, err := NewDatabasePool(nil)

	if err == nil {
		t.Error("Expected error due to empty DSN, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestNewDatabasePool_WithCustomConfig(t *testing.T) {
	config := &PoolConfig{
		DSN:             "invalid://connection:string",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute * 30,
		ConnMaxIdleTime: time.Minute * 15,
		LogLevel:        logger.Silent,
	}

	_, err := NewDatabasePool(config)

	if err == nil {
		t.Error("Expected error due to invalid DSN, got nil")
	}
}

func TestDatabasePool_Stats_WithoutConnection(t *testing.T) {
	pool := &DatabasePool{
		DB: nil,
		config: &PoolConfig{
			MaxOpenConns: 10,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stats() should handle nil DB gracefully, but got panic: %v", r)
		}
	}()

	stats := pool.Stats()

	if _, hasError := stats["error"]; !hasError {
		t.Error("Expected error in stats when DB is nil")
	}
}

func TestDatabasePool_Health_WithoutConnection(t *testing.T) {
	pool := &DatabasePool{
		DB: nil,
	}

	err := pool.Health()

	if err == nil {
		t.Error("Expected error when checking health with nil DB")
	}
}

func TestDatabasePool_Close_WithoutConnection(t *testing.T) {
	pool := &DatabasePool{
		DB: nil,
	}

	err := pool.Close()

	if err != nil {
		t.Errorf("Expected no error when closing nil DB, got: %v", err)
	}
}

func TestPoolConfig_Validation(t *testing.T) {
	tests := []struct {
		name     string
		config   *PoolConfig
		expected bool
	}{
		{
			name: "Valid configuration",
			config: &PoolConfig{
				DSN:             "postgres://user:pass@localhost/dbname",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: time.Minute * 30,
				LogLevel:        logger.Info,
			},
			expected: true,
		},
		{
			name: "Zero values configuration",
			config: &PoolConfig{
				DSN:             "",
				MaxOpenConns:    0,
				MaxIdleConns:    0,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: 0,
				LogLevel:        logger.Silent,
			},
			expected: false,
		},
		{
			name: "Negative values configuration",
			config: &PoolConfig{
				DSN:             "postgres://user:pass@localhost/dbname",
				MaxOpenConns:    -1,
				MaxIdleConns:    -1,
				ConnMaxLifetime: -time.Hour,
				ConnMaxIdleTime: -time.Minute,
				LogLevel:        logger.Info,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDatabasePool(tt.config)

			if tt.expected && err == nil {
				t.Error("Expected successful pool creation but got error:", err)
			} else if !tt.expected && err == nil {
				t.Error("Expected error but pool creation succeeded")
			}
		})
	}
}

func BenchmarkDefaultPoolConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultPoolConfig()
	}
}

func BenchmarkPoolConfig_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := &PoolConfig{
			DSN:             "postgres://user:pass@localhost/dbname",
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: time.Minute * 30,
			LogLevel:        logger.Info,
		}
		_ = config
	}
}
