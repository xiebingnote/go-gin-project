package config

// PostgresqlConfigEntry Postgresql config entry
type PostgresqlConfigEntry struct {
	Postgresql struct {
		Host     string `toml:"Host"`     // host
		Port     int    `toml:"Port"`     // port
		User     string `toml:"User"`     // username
		Password string `toml:"PassWord"` // password
		DBName   string `toml:"DBName"`   // dbname
		SSLMode  string `toml:"SSLMode"`  // ssl mode
	} `toml:"Postgresql"`
	Pool       PoolConfig       `toml:"Pool"`       // pool
	Migrations MigrationsConfig `toml:"Migrations"` // migrations
}

// PoolConfig Pool config
type PoolConfig struct {
	MaxConns          int `toml:"MaxConns"`          // max connections
	MinConns          int `toml:"MinConns"`          // min connections
	MaxConnLifetime   int `toml:"MaxConnLifetime"`   // max connection lifetime
	MaxConnIdleTime   int `toml:"MaxConnIdleTime"`   // max connection idle time
	HealthCheckPeriod int `toml:"HealthCheckPeriod"` // health check period
}

// MigrationsConfig Migrations config
type MigrationsConfig struct {
	Path  string `toml:"Path"`  // migration path
	Table string `toml:"Table"` // migration table
}
