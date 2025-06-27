package config

import "time"

type MongoDBConfigEntry struct {
	Mongo struct {
		Host            string        `toml:"Host"`            // MongoDB server host
		Port            int64         `toml:"Port"`            // MongoDB server port
		Username        string        `toml:"UserName"`        // MongoDB username
		Password        string        `toml:"PassWord"`        // MongoDB password
		DBName          string        `toml:"DBName"`          // MongoDB database name
		ConnectTimeout  time.Duration `toml:"ConnectTimeout"`  // Connection timeout in milliseconds
		ServerTimeout   time.Duration `toml:"ServerTimeout"`   // Server selection timeout in milliseconds
		MaxPoolSize     uint64        `toml:"MaxPoolSize"`     // Maximum connection pool size
		MinPoolSize     uint64        `toml:"MinPoolSize"`     // Minimum connection pool size
		MaxConnIdleTime time.Duration `toml:"MaxConnIdleTime"` // Maximum connection idle time in milliseconds
	} `toml:"MongoDB"`
}
