package config

import (
	"time"

	"github.com/ulule/limiter/v3"
)

var (
	// PublicRate 公共API限流规则（按IP）
	PublicRate = limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}

	// AuthUserRate 认证用户限流规则（按UserID）
	AuthUserRate = limiter.Rate{
		Period: 1 * time.Hour,
		Limit:  1000,
	}

	// LoginRate 敏感端点限流（登录尝试）
	LoginRate = limiter.Rate{
		Period: 5 * time.Minute,
		Limit:  10,
	}
)
