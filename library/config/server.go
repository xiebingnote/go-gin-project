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

	// 新增的服务器选项配置
	Options ServerOptions `toml:"Options"`
}

// ServerOptions 服务器配置选项
type ServerOptions struct {
	// 基础配置
	Mode           string   `toml:"Mode"`           // gin模式: debug, release, test
	EnablePprof    bool     `toml:"EnablePprof"`    // 是否启用pprof
	EnableMetrics  bool     `toml:"EnableMetrics"`  // 是否启用metrics
	TrustedProxies []string `toml:"TrustedProxies"` // 信任的代理IP

	// 安全配置
	EnableCORS      bool                   `toml:"EnableCORS"`     // 是否启用CORS
	EnableSecurity  bool                   `toml:"EnableSecurity"` // 是否启用安全头
	RateLimitConfig *ServerRateLimitConfig `toml:"RateLimit"`      // 限流配置

	// 认证配置
	AuthType   string `toml:"AuthType"`   // 认证类型: jwt, casbin
	EnableAuth bool   `toml:"EnableAuth"` // 是否启用认证

	// 监控配置
	EnableHealthCheck bool   `toml:"EnableHealthCheck"` // 是否启用健康检查
	HealthCheckPath   string `toml:"HealthCheckPath"`   // 健康检查路径

	// 超时配置
	ReadTimeout     time.Duration `toml:"ReadTimeout"`     // 读取超时
	WriteTimeout    time.Duration `toml:"WriteTimeout"`    // 写入超时
	IdleTimeout     time.Duration `toml:"IdleTimeout"`     // 空闲超时
	ShutdownTimeout time.Duration `toml:"ShutdownTimeout"` // 关闭超时
}

// ServerRateLimitConfig 服务器限流配置
type ServerRateLimitConfig struct {
	EnableRedis  bool `toml:"EnableRedis"`  // 是否使用Redis限流
	EnableMemory bool `toml:"EnableMemory"` // 是否使用内存限流
	LoginLimit   int  `toml:"LoginLimit"`   // 登录限流次数
	APILimit     int  `toml:"APILimit"`     // API限流次数
	PublicLimit  int  `toml:"PublicLimit"`  // 公共API限流次数
}
