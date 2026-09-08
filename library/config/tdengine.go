package config

import "time"

type TDengineConfigEntry struct {
	TDengine struct {
		Protocol        string        `toml:"Protocol"`        // ws or wss (taosAdapter)
		Host            string        `toml:"Host"`            // TDengine server host
		Port            int64         `toml:"Port"`            // TDengine server port
		UserName        string        `toml:"UserName"`        // TDengine username
		PassWord        string        `toml:"PassWord"`        // TDengine password
		Database        string        `toml:"Database"`        // TDengine database name
		ConnectTimeout  time.Duration `toml:"ConnectTimeout"`  // Connection timeout (Go duration string, e.g. "10s")
		ReadTimeout     time.Duration `toml:"ReadTimeout"`     // Read timeout (Go duration string, e.g. "10s")
		WriteTimeout    time.Duration `toml:"WriteTimeout"`    // Write timeout (Go duration string, e.g. "10s")
		MaxOpenConns    int           `toml:"MaxOpenConns"`    // Maximum number of open connections
		MaxIdleConns    int           `toml:"MaxIdleConns"`    // Maximum number of idle connections
		ConnMaxLifetime time.Duration `toml:"ConnMaxLifetime"` // Maximum connection lifetime (Go duration string, e.g. "10s")
		ConnMaxIdleTime time.Duration `toml:"ConnMaxIdleTime"` // Maximum connection idle time (Go duration string, e.g. "10s")
	} `toml:"TDengine"`
}
