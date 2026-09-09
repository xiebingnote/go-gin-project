package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"go.uber.org/zap"
)

func MemoryLimiter(rate limiter.Rate) gin.HandlerFunc {
	return limitergin.NewMiddleware(limiter.New(memory.NewStore(), rate))
}

func RedisLimiter(rate limiter.Rate) gin.HandlerFunc {
	return redisIPLimiter("rate_limit:public:", rate)
}

// UserIDLimiter keeps a single counter store for the lifetime of this middleware.
func UserIDLimiter(rate limiter.Rate) gin.HandlerFunc {
	instance := limiter.New(memory.NewStore(), rate)
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists || userID == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		state, err := instance.Get(c.Request.Context(), fmt.Sprintf("user:%v", userID))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter unavailable"})
			return
		}
		if state.Reached {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func IPWhitelist(whitelist []string) gin.HandlerFunc {
	return IPWhitelistMiddleware(whitelist)
}

// Optional rates preserve existing callers while allowing per-server configuration.
func LoginRateLimiter(rates ...limiter.Rate) gin.HandlerFunc {
	rate := config.LoginRate
	if len(rates) > 0 {
		rate = rates[0]
	}
	return redisIPLimiter("rate_limit:login:", rate)
}

func APIRateLimiter(rates ...limiter.Rate) gin.HandlerFunc {
	rate := config.APIRate
	if len(rates) > 0 {
		rate = rates[0]
	}
	return redisIPLimiter("rate_limit:api:", rate)
}

func redisIPLimiter(prefix string, rate limiter.Rate) gin.HandlerFunc {
	if rate.Limit <= 0 || rate.Period <= 0 {
		panic("rate limit and period must be positive")
	}
	client := resource.RedisClient
	if client == nil {
		// A missing Redis client must not disable protection or panic on requests.
		return MemoryLimiter(rate)
	}
	seconds := int64(rate.Period / time.Second)
	if rate.Period%time.Second != 0 {
		seconds++
	}
	return func(c *gin.Context) {
		key := prefix + c.ClientIP()
		allowed, err := client.Eval(c.Request.Context(), config.LuaScript, []string{key}, rate.Limit, seconds).Int()
		if err != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error("Rate limiter failed", zap.Error(err))
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Rate limiter unavailable"})
			return
		}
		if allowed == 0 {
			c.Header("Retry-After", strconv.FormatInt(seconds, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}
		c.Next()
	}
}
