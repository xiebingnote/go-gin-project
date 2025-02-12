package config

import (
	"time"
)

// ServerConfigEntry 服务配置
type ServerConfigEntry struct {
	HTTPServer struct {
		Listen       string        `toml:"Listen"`       // 监听地址
		ReadTimeout  time.Duration `toml:"ReadTimeout"`  // 单位：毫秒
		WriteTimeout time.Duration `toml:"WriteTimeout"` // 单位：毫秒
		IdleTimeout  time.Duration `toml:"IdleTimeout"`  // 单位：毫秒
	} `toml:"HTTPServer"`

	AdminServer struct {
		Listen string `toml:"Listen"` // 监听地址
	} `toml:"AdminServer"`

	Version struct {
		Version string `toml:"Version"` // 版本号
	} `toml:"Version"`
}
