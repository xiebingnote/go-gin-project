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

// LuaScript 脚本（用于限流）
const LuaScript = `
	local key = KEYS[1]
	local limit = tonumber(ARGV[1])
	local expireTime = tonumber(ARGV[2])

	local current = redis.call("INCR", key)
	if current == 1 then
		redis.call("EXPIRE", key, expireTime)
	end

	if current > limit then
		return 0
	end
	return 1
`
