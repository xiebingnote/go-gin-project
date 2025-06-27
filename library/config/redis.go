package config

import "time"

// RedisConfigEntry is the configuration entry for redis.
type RedisConfigEntry struct {
	Redis struct {
		Addr            string        `toml:"Addr"`            // Redis server address
		Password        string        `toml:"Password"`        // Redis password
		DB              int           `toml:"DB"`              // Redis database number
		PoolSize        int           `toml:"PoolSize"`        // Maximum number of connections in pool
		MinIdleConns    int           `toml:"MinIdleConns"`    // Minimum number of idle connections
		MaxRetries      int           `toml:"MaxRetries"`      // Maximum number of retries
		DialTimeout     time.Duration `toml:"DialTimeout"`     // Connection timeout in milliseconds
		ReadTimeout     time.Duration `toml:"ReadTimeout"`     // Read timeout in milliseconds
		WriteTimeout    time.Duration `toml:"WriteTimeout"`    // Write timeout in milliseconds
		PoolTimeout     time.Duration `toml:"PoolTimeout"`     // Pool timeout in milliseconds
		IdleTimeout     time.Duration `toml:"IdleTimeout"`     // Idle connection timeout in milliseconds
		MaxConnAge      time.Duration `toml:"MaxConnAge"`      // Maximum connection age in milliseconds
		IdleCheckFreq   time.Duration `toml:"IdleCheckFreq"`   // Idle check frequency in milliseconds
		HealthCheckFreq time.Duration `toml:"HealthCheckFreq"` // Health check frequency in milliseconds
	} `toml:"Redis"`
}
