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
func InitRedis(_ context.Context) {
	if err := InitRedisClient(); err != nil {
		// Log an error message if the Redis connection cannot be established.
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize Redis: %v", err))
		panic(err.Error())
	}
}

// InitRedisClient initializes the Redis client using the global RedisConfig.
// It sets up the Redis client with the configuration options specified in the
// RedisConfig and tests the connection by pinging the Redis server.
//
// The Redis client is configured with the following options:
//
// - Addr: The address of the Redis server.
// - Password: The password to use for authentication.
// - DB: The number of the Redis database to use.
// - PoolSize: The maximum number of connections to keep in the connection pool.
// - MinIdleConns: The minimum number of idle connections to keep in the connection pool.
// - MaxRetries: The maximum number of times to retry a failed operation.
// - DialTimeout: The timeout for establishing a connection to the Redis server.
// - ReadTimeout: The timeout for reading from the Redis server.
// - WriteTimeout: The timeout for writing to the Redis server.
// - PoolTimeout: The maximum wait time for a connection from the pool.
//
// Returns an error if the connection to the Redis server cannot be established.
func InitRedisClient() error {
	// Retrieve the Redis configuration
	cfg := config.RedisConfig

	// Validate the Redis configuration
	if err := validateRedisConfig(cfg); err != nil {
		return fmt.Errorf("invalid Redis configuration: %w", err)
	}

	// Initialize the Redis client with the specified options
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Millisecond,
		PoolTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Millisecond, // Use ReadTimeout as pool timeout
	})

	// Create a context with timeout for the ping operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the Redis server
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to ping Redis server: %w", err)
	}

	// Store the Redis client instance in the global resource
	resource.RedisClient = redisClient
	return nil
}

// validateRedisConfig validates the Redis configuration.
//
// The function checks the following:
//
//  1. The Redis configuration is not nil.
//  2. The address is not empty.
//  3. The connection pool settings are valid.
//  4. The timeout settings are valid.
//
// If the configuration is invalid, the function returns an error.
func validateRedisConfig(cfg *config.RedisConfigEntry) error {
	// Check for nil configuration
	if cfg == nil {
		return fmt.Errorf("redis configuration is nil")
	}

	// Check for empty address
	if cfg.Redis.Addr == "" {
		return fmt.Errorf("redis address is empty")
	}

	// Check for invalid pool size
	if cfg.Redis.PoolSize <= 0 {
		return fmt.Errorf("invalid pool size: %d, must be greater than 0", cfg.Redis.PoolSize)
	}

	// Check for invalid minimum idle connections
	if cfg.Redis.MinIdleConns < 0 {
		return fmt.Errorf("invalid minimum idle connections: %d, must be non-negative", cfg.Redis.MinIdleConns)
	}

	// Check for invalid max retries
	if cfg.Redis.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries: %d, must be non-negative", cfg.Redis.MaxRetries)
	}

	// Check for invalid timeout settings
	if cfg.Redis.DialTimeout <= 0 {
		return fmt.Errorf("invalid dial timeout: %d ms, must be greater than 0", cfg.Redis.DialTimeout)
	}
	if cfg.Redis.ReadTimeout <= 0 {
		return fmt.Errorf("invalid read timeout: %d ms, must be greater than 0", cfg.Redis.ReadTimeout)
	}
	if cfg.Redis.WriteTimeout <= 0 {
		return fmt.Errorf("invalid write timeout: %d ms, must be greater than 0", cfg.Redis.WriteTimeout)
	}

	// Check for logical consistency
	if cfg.Redis.MinIdleConns > cfg.Redis.PoolSize {
		return fmt.Errorf("minimum idle connections (%d) cannot be greater than pool size (%d)",
			cfg.Redis.MinIdleConns, cfg.Redis.PoolSize)
	}

	return nil
}

// CloseRedis closes the Redis client connection.
//
// This function checks if the global RedisClient resource is initialized.
// If it is, it attempts to close the client connection and returns an error
// if the closure fails. If successful, it returns nil.
//
// Returns:
//   - An error if the client close operation fails.
//   - nil if the Redis client is nil or the connection is closed successfully.
func CloseRedis() error {
	if resource.RedisClient == nil {
		// Redis client is nil, no connection to close
		return nil
	}

	// Attempt to close the Redis client connection
	if err := resource.RedisClient.Close(); err != nil {
		// Return an error if closing the connection fails
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	// Reset the global Redis client to nil
	resource.RedisClient = nil
	return nil
}
