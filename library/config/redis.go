package config

// RedisConfigEntry is the configuration entry for redis.
type RedisConfigEntry struct {
	Redis struct {
		Addr         string `toml:"Addr"`         // redis address
		Password     string `toml:"Password"`     // redis password
		DB           int    `toml:"DB"`           // redis db
		PoolSize     int    `toml:"PoolSize"`     // redis pool size
		MinIdleConns int    `toml:"MinIdleConns"` // redis min idle conns
		MaxRetries   int    `toml:"MaxRetries"`   // redis max retries
		DialTimeout  int    `toml:"DialTimeout"`  // 单位：ms
		ReadTimeout  int    `toml:"ReadTimeout"`  // 单位：ms
		WriteTimeout int    `toml:"WriteTimeout"` // 单位：ms
	} `toml:"Redis"`
}
