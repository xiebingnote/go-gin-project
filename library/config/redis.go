package config

type RedisConfigEntry struct {
	Redis struct {
		Addr         string `toml:"Addr"`
		Password     string `toml:"Password"`
		DB           int    `toml:"DB"`
		PoolSize     int    `toml:"PoolSize"`
		MinIdleConns int    `toml:"MinIdleConns"`
		MaxRetries   int    `toml:"MaxRetries"`
		DialTimeout  int    `toml:"DialTimeout"`  // 单位：ms
		ReadTimeout  int    `toml:"ReadTimeout"`  // 单位：ms
		WriteTimeout int    `toml:"WriteTimeout"` // 单位：ms
	} `toml:"Redis"`
}
