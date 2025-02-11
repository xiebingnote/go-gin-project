package config

import "time"

type PostgresqlConfigEntry struct {
	Host       string           `toml:"Host"`
	Port       int              `toml:"Port"`
	User       string           `toml:"User"`
	Password   string           `toml:"PassWord"`
	DBName     string           `toml:"DBName"`
	SSLMode    string           `toml:"SSLMode"`
	Pool       PoolConfig       `toml:"Pool"`
	Migrations MigrationsConfig `toml:"Migrations"`
}

type PoolConfig struct {
	MaxConns          int           `toml:"MaxConns"`
	MinConns          int           `toml:"MinConns"`
	MaxConnLifetime   time.Duration `toml:"MaxConnLifetime"`
	MaxConnIdleTime   time.Duration `toml:"MaxConnIdleTime"`
	HealthCheckPeriod time.Duration `toml:"HealthCheckPeriod"`
}

type MigrationsConfig struct {
	Path  string `toml:"Path"`
	Table string `toml:"Table"`
}
