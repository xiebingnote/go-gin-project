package config

import "time"

type TDengineConfigEntry struct {
	TDengine struct {
		Host            string        `toml:"Host"`            // TDengine server host
		Port            int64         `toml:"Port"`            // TDengine server port
		UserName        string        `toml:"UserName"`        // TDengine username
		PassWord        string        `toml:"PassWord"`        // TDengine password
		Database        string        `toml:"Database"`        // TDengine database name
		ConnectTimeout  time.Duration `toml:"ConnectTimeout"`  // Connection timeout in milliseconds
		ReadTimeout     time.Duration `toml:"ReadTimeout"`     // Read timeout in milliseconds
		WriteTimeout    time.Duration `toml:"WriteTimeout"`    // Write timeout in milliseconds
		MaxOpenConns    int           `toml:"MaxOpenConns"`    // Maximum number of open connections
		MaxIdleConns    int           `toml:"MaxIdleConns"`    // Maximum number of idle connections
		ConnMaxLifetime time.Duration `toml:"ConnMaxLifetime"` // Maximum connection lifetime in milliseconds
		ConnMaxIdleTime time.Duration `toml:"ConnMaxIdleTime"` // Maximum connection idle time in milliseconds
	} `toml:"TDengine"`
}
