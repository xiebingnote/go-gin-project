package service

import (
	"context"
	"fmt"
	"time"

	"project/library/config"
	"project/library/resource"

	"github.com/redis/go-redis/v9"
)

// InitRedis initializes the Redis database connection.
//
// This function reads the Redis configuration from the global RedisConfig,
// initializes the Redis client using the configuration, and assigns the
// client to the global RedisClient resource.
func InitRedis(_ context.Context) {
	err := InitRedisClient()
	if err != nil {
		// Panic if the Redis client cannot be initialized.
		panic(err.Error())
	}
}

// InitRedisClient initializes the Redis client using the global RedisConfig.
// It sets up the Redis client with the configuration options specified in the
// RedisConfig and tests the connection by pinging the Redis server.
//
// Returns an error if the connection to the Redis server cannot be established.
func InitRedisClient() error {
	// Retrieve the Redis configuration
	cfg := config.RedisConfig

	// Initialize the Redis client with the specified options
	RedisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Millisecond,
	})

	// Test the connection to the Redis server with a ping
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	resource.RedisClient = RedisClient
	return nil
}
