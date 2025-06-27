package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForRedis initializes a test logger for testing purposes
func setupTestLoggerForRedis() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestRedisConfig initializes a test configuration for Redis
func setupTestRedisConfig() {
	config.RedisConfig = &config.RedisConfigEntry{}
	config.RedisConfig.Redis.Addr = "127.0.0.1:6379"
	config.RedisConfig.Redis.Password = ""
	config.RedisConfig.Redis.DB = 0
	config.RedisConfig.Redis.PoolSize = 10
	config.RedisConfig.Redis.MinIdleConns = 5
	config.RedisConfig.Redis.MaxRetries = 3
	config.RedisConfig.Redis.DialTimeout = 5000 * time.Millisecond
	config.RedisConfig.Redis.ReadTimeout = 3000 * time.Millisecond
	config.RedisConfig.Redis.WriteTimeout = 3000 * time.Millisecond
	config.RedisConfig.Redis.PoolTimeout = 4000 * time.Millisecond
	config.RedisConfig.Redis.IdleTimeout = 300000 * time.Millisecond
	config.RedisConfig.Redis.MaxConnAge = 0
	config.RedisConfig.Redis.IdleCheckFreq = 60000 * time.Millisecond
	config.RedisConfig.Redis.HealthCheckFreq = 30000 * time.Millisecond
}

// TestValidateRedisConfig tests the validation of Redis configuration settings.
//
// It sets up a series of test cases to verify that the validateRedisConfig function
// correctly identifies invalid configurations and produces the expected error messages.
// The test cases include various scenarios such as nil configuration, empty address,
// invalid database number, and other invalid settings related to pool size, timeouts,
// and connection frequencies. Each test case specifies a setup function to configure
// the Redis settings and expects either an error or no error with a specific error message.
func TestValidateRedisConfig(t *testing.T) {
	setupTestLoggerForRedis()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.RedisConfig = nil
			},
			expectError: true,
			errorMsg:    "redis configuration is not initialized",
		},
		{
			name: "empty address",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.Addr = ""
			},
			expectError: true,
			errorMsg:    "redis address is not configured",
		},
		{
			name: "invalid database number",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.DB = -1
			},
			expectError: true,
			errorMsg:    "invalid database number: -1, must be non-negative",
		},
		{
			name: "invalid pool size",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.PoolSize = 0
			},
			expectError: true,
			errorMsg:    "invalid pool size: 0, must be greater than 0",
		},
		{
			name: "invalid min idle connections",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.MinIdleConns = -1
			},
			expectError: true,
			errorMsg:    "invalid minimum idle connections: -1, must be non-negative",
		},
		{
			name: "invalid max retries",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.MaxRetries = -1
			},
			expectError: true,
			errorMsg:    "invalid max retries: -1, must be non-negative",
		},
		{
			name: "invalid dial timeout",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.DialTimeout = 0
			},
			expectError: true,
			errorMsg:    "invalid dial timeout: 0s, must be greater than 0",
		},
		{
			name: "invalid read timeout",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.ReadTimeout = 0
			},
			expectError: true,
			errorMsg:    "invalid read timeout: 0s, must be greater than 0",
		},
		{
			name: "invalid write timeout",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.WriteTimeout = 0
			},
			expectError: true,
			errorMsg:    "invalid write timeout: 0s, must be greater than 0",
		},
		{
			name: "invalid pool timeout",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.PoolTimeout = 0
			},
			expectError: true,
			errorMsg:    "invalid pool timeout: 0s, must be greater than 0",
		},
		{
			name: "min idle conns greater than pool size",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.PoolSize = 5
				config.RedisConfig.Redis.MinIdleConns = 10
			},
			expectError: true,
			errorMsg:    "minimum idle connections (10) cannot be greater than pool size (5)",
		},
		{
			name: "invalid idle timeout",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.IdleTimeout = -1 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "invalid idle timeout: -1ms, must be non-negative",
		},
		{
			name: "invalid max connection age",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.MaxConnAge = -1 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "invalid max connection age: -1ms, must be non-negative",
		},
		{
			name: "invalid idle check frequency",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.IdleCheckFreq = -1 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "invalid idle check frequency: -1ms, must be non-negative",
		},
		{
			name: "invalid health check frequency",
			setupConfig: func() {
				setupTestRedisConfig()
				config.RedisConfig.Redis.HealthCheckFreq = -1 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "invalid health check frequency: -1ms, must be non-negative",
		},
		{
			name: "valid config",
			setupConfig: func() {
				setupTestRedisConfig()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			err := validateRedisConfig(config.RedisConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestCloseRedis_NoClient tests that CloseRedis returns no error when called with no
// Redis client set.
func TestCloseRedis_NoClient(t *testing.T) {
	setupTestLoggerForRedis()

	// Ensure no client is set
	resource.RedisClient = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseRedis(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no client, got: %v", err)
	}
}

// TestInitRedis_WithValidConfig tests that InitRedis does not panic when called with
// a valid Redis configuration.
//
// This test will only pass if Redis is actually running.
// If Redis is not available, this test will be skipped.
func TestInitRedis_WithValidConfig(t *testing.T) {
	setupTestLoggerForRedis()
	setupTestRedisConfig()

	// This test will only pass if Redis is actually running
	// Skip if Redis is not available
	t.Skip("Skipping integration test - requires running Redis instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitRedis panicked: %v", r)
		}
	}()

	InitRedis(ctx)

	// Clean up
	if resource.RedisClient != nil {
		_ = CloseRedis(ctx)
	}
}
