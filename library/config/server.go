package config

import (
	"time"
)

type ServerConfigEntry struct {
	HTTPServer struct {
		Listen       string        `toml:"Listen"`
		ReadTimeout  time.Duration `toml:"ReadTimeout"`  // 单位：毫秒
		WriteTimeout time.Duration `toml:"WriteTimeout"` // 单位：毫秒
		IdleTimeout  time.Duration `toml:"IdleTimeout"`  // 单位：毫秒
	} `toml:"HTTPServer"`
	AdminServer struct {
		Listen string `toml:"Listen"`
	} `toml:"AdminServer"`
}
