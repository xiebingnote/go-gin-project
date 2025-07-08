package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/redis/go-redis/v9"
)

// InitRedis initializes the Redis database connection.
//
// This function reads the Redis configuration from the global RedisConfig,
// initializes the Redis client using the configuration, and assigns the
// client to the global RedisClient resource.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
func InitRedis(ctx context.Context) {
	if err := InitRedisClient(ctx); err != nil {
		// Log an error message if the Redis connection cannot be established.
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize Redis: %v", err))
		panic(fmt.Sprintf("Redis initialization failed: %v", err))
	}
}

// InitRedisClient initializes the Redis client using the global RedisConfig.
// It sets up the Redis client with the configuration options specified in the
// RedisConfig and tests the connection by pinging the Redis server.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates the Redis configuration
// 2. Creates Redis client with comprehensive configuration
// 3. Tests the connection with ping and info commands
// 4. Performs health checks if configured
// 5. Stores the client in global resource
func InitRedisClient(ctx context.Context) error {
	// Validate the Redis configuration
	if err := validateRedisConfig(config.RedisConfig); err != nil {
		return fmt.Errorf("redis configuration validation failed: %w", err)
	}

	cfg := &config.RedisConfig.Redis

	resource.LoggerService.Info(fmt.Sprintf("initializing redis client for address: %s", cfg.Addr))

	// Initialize the Redis client with comprehensive configuration
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,

		// Timeout configurations with proper time unit conversion
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,

		// Advanced connection pool configurations
		ConnMaxIdleTime: cfg.IdleTimeout,
		ConnMaxLifetime: cfg.MaxConnAge,
	})

	// Perform comprehensive connection testing
	if err := testRedisConnection(ctx, redisClient); err != nil {
		// Clean up the client if connection test fails
		if closeErr := redisClient.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close redis client during cleanup: %v", closeErr))
		}
		return fmt.Errorf("redis connection test failed: %w", err)
	}

	// Store the Redis client instance in the global resource
	resource.RedisClient = redisClient

	// Start health check if configured
	if cfg.HealthCheckFreq > 0 {
		go startRedisHealthCheck(ctx, cfg.HealthCheckFreq)
	}

	resource.LoggerService.Info(fmt.Sprintf("✅ successfully initialized redis client for address: %s, DB: %d",
		cfg.Addr, cfg.DB))

	return nil
}

// testRedisConnection performs comprehensive connection testing for Redis.
//
// Parameters:
//   - ctx: Context for the operation
//   - client: Redis client to test
//
// Returns:
//   - error: An error if any test fails, nil otherwise
func testRedisConnection(ctx context.Context, client *redis.Client) error {
	// Create timeout context for connection tests
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resource.LoggerService.Info("testing redis connection")

	// Test 1: Basic ping
	if _, err := client.Ping(testCtx).Result(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("redis ping test failed: %v", err))
		return fmt.Errorf("ping test failed: %w", err)
	}

	// Test 2: Get server info
	info, err := client.Info(testCtx).Result()
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("redis info command failed: %v", err))
		return fmt.Errorf("info command failed: %w", err)
	}

	// Log server information
	resource.LoggerService.Info("redis server info retrieved successfully")

	// Test 3: Basic set/get operation
	testKey := "redis:health:test"
	testValue := "test_value"

	if err := client.Set(testCtx, testKey, testValue, time.Minute).Err(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("redis set test failed: %v", err))
		return fmt.Errorf("set test failed: %w", err)
	}

	val, err := client.Get(testCtx, testKey).Result()
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("redis get test failed: %v", err))
		return fmt.Errorf("get test failed: %w", err)
	}

	if val != testValue {
		resource.LoggerService.Error(fmt.Sprintf("redis value mismatch: expected %s, got %s", testValue, val))
		return fmt.Errorf("value mismatch: expected %s, got %s", testValue, val)
	}

	// Clean up test key
	if err := client.Del(testCtx, testKey).Err(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to clean up test key: %v", err))
		// Don't return error for cleanup failure
	}

	// Parse and log useful server information
	if len(info) > 0 {
		resource.LoggerService.Info("redis connection test completed successfully")
	}

	return nil
}

// startRedisHealthCheck starts a background health check for Redis.
//
// Parameters:
//   - ctx: Context for the operation
//   - interval: Health check interval
func startRedisHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	resource.LoggerService.Info(fmt.Sprintf("starting redis health check with interval: %v", interval))

	for {
		select {
		case <-ctx.Done():
			resource.LoggerService.Info("redis health check stopped")
			return
		case <-ticker.C:
			if resource.RedisClient != nil {
				healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if _, err := resource.RedisClient.Ping(healthCtx).Result(); err != nil {
					resource.LoggerService.Error(fmt.Sprintf("redis health check failed: %v", err))
				}
				cancel()
			}
		}
	}
}

// validateRedisConfig validates the Redis configuration.
//
// Parameters:
//   - cfg: Redis configuration to validate
//
// Returns:
//   - error: An error if configuration is invalid, nil otherwise
//
// The function validates:
// 1. Configuration is not nil
// 2. Address is not empty
// 3. Connection pool settings are valid
// 4. Timeout settings are valid
// 5. Logical consistency between settings
func validateRedisConfig(cfg *config.RedisConfigEntry) error {
	// Check for nil configuration
	if cfg == nil {
		return fmt.Errorf("redis configuration is not initialized")
	}

	redisCfg := &cfg.Redis

	// Check for empty address
	if redisCfg.Addr == "" {
		return fmt.Errorf("redis address is not configured")
	}

	// Validate database number
	if redisCfg.DB < 0 {
		return fmt.Errorf("invalid database number: %d, must be non-negative", redisCfg.DB)
	}

	// Check for invalid pool size
	if redisCfg.PoolSize <= 0 {
		return fmt.Errorf("invalid pool size: %d, must be greater than 0", redisCfg.PoolSize)
	}

	// Check for invalid minimum idle connections
	if redisCfg.MinIdleConns < 0 {
		return fmt.Errorf("invalid minimum idle connections: %d, must be non-negative", redisCfg.MinIdleConns)
	}

	// Check for invalid max retries
	if redisCfg.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries: %d, must be non-negative", redisCfg.MaxRetries)
	}

	// Check for invalid timeout settings
	if redisCfg.DialTimeout <= 0 {
		return fmt.Errorf("invalid dial timeout: %v, must be greater than 0", redisCfg.DialTimeout)
	}
	if redisCfg.ReadTimeout <= 0 {
		return fmt.Errorf("invalid read timeout: %v, must be greater than 0", redisCfg.ReadTimeout)
	}
	if redisCfg.WriteTimeout <= 0 {
		return fmt.Errorf("invalid write timeout: %v, must be greater than 0", redisCfg.WriteTimeout)
	}
	if redisCfg.PoolTimeout <= 0 {
		return fmt.Errorf("invalid pool timeout: %v, must be greater than 0", redisCfg.PoolTimeout)
	}

	// Check for logical consistency
	if redisCfg.MinIdleConns > redisCfg.PoolSize {
		return fmt.Errorf("minimum idle connections (%d) cannot be greater than pool size (%d)",
			redisCfg.MinIdleConns, redisCfg.PoolSize)
	}

	// Validate optional timeout settings (0 means disabled)
	if redisCfg.IdleTimeout < 0 {
		return fmt.Errorf("invalid idle timeout: %v, must be non-negative", redisCfg.IdleTimeout)
	}
	if redisCfg.MaxConnAge < 0 {
		return fmt.Errorf("invalid max connection age: %v, must be non-negative", redisCfg.MaxConnAge)
	}
	if redisCfg.IdleCheckFreq < 0 {
		return fmt.Errorf("invalid idle check frequency: %v, must be non-negative", redisCfg.IdleCheckFreq)
	}
	if redisCfg.HealthCheckFreq < 0 {
		return fmt.Errorf("invalid health check frequency: %v, must be non-negative", redisCfg.HealthCheckFreq)
	}

	return nil
}

// CloseRedis closes the Redis client connection gracefully.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if Redis client is initialized
// 2. Attempts to close the connection with timeout
// 3. Clears the global resource reference
func CloseRedis(ctx context.Context) error {
	if resource.RedisClient == nil {
		return nil
	}

	if resource.LoggerService != nil {
		resource.LoggerService.Info("closing redis client connection")
	}

	// Create timeout context for close operation
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Attempt to perform a final ping to check connection status
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- resource.RedisClient.Close()
	}()

	// Wait for close operation or timeout
	select {
	case err := <-done:
		if err != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error(fmt.Sprintf("failed to close redis connection: %v", err))
			}
			return fmt.Errorf("failed to close redis connection: %w", err)
		}
	case <-closeCtx.Done():
		if resource.LoggerService != nil {
			resource.LoggerService.Error("redis close operation timeout")
		}
		return fmt.Errorf("redis close operation timeout")
	}

	// Clear the global Redis client reference
	resource.RedisClient = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 successfully closed redis client connection")
	}

	return nil
}
