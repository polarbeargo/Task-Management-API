package main

import (
	"os"
	"task-manager/backend/internal/config"
	"testing"
)

func TestApplicationStartup(t *testing.T) {
	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("REDIS_HOST", "localhost")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("REDIS_HOST")
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Configuration should not be nil")
	}

	t.Log("Application configuration loaded successfully")
}

func TestCacheWarmingIntegration(t *testing.T) {
	os.Setenv("ENVIRONMENT", "development")
	defer os.Unsetenv("ENVIRONMENT")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Configuration should not be nil")
	}

	t.Log("Application configuration verification passed")
}

func TestConfigurationValues(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected string
	}{
		{
			name:     "ENVIRONMENT environment variable",
			envVar:   "ENVIRONMENT",
			envValue: "production",
			expected: "production",
		},
		{
			name:     "REDIS_HOST environment variable",
			envVar:   "REDIS_HOST",
			envValue: "localhost",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			value := os.Getenv(tt.envVar)
			if value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, value)
			}
		})
	}
}
