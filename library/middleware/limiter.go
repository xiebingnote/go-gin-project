package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ulule/limiter/v3"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/ulule/limiter/v3/drivers/store/redis"
)

// MemoryLimiter creates a middleware that limits the number of requests to a
// given rate. It uses an in-memory store to keep track of the request counts.
// The rate is specified as a limiter.Rate object.
func MemoryLimiter(rate limiter.Rate) gin.HandlerFunc {
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return limitergin.NewMiddleware(instance)
}

// RedisLimiter creates a rate limiting middleware using Redis as the storage backend.
//
// This middleware limits requests based on a specified rate and uses a combination
// of client IP and user ID (if available) as the key for rate limiting. It trusts
// the "X-Forwarded-For" header for IP resolution when behind a proxy.
//
// Parameters:
//   - rate: The rate limit configuration (limiter.Rate).
//
// Returns:
//   - gin.HandlerFunc: The Gin middleware function for rate limiting.
func RedisLimiter(rate limiter.Rate) gin.HandlerFunc {
	// Create a Redis store for rate limiting with specified options.
	store, err := redis.NewStoreWithOptions(resource.RedisClient, limiter.StoreOptions{
		Prefix:          "limiter:",       // Prefix for keys in Redis.
		CleanUpInterval: 30 * time.Minute, // Interval for cleaning up expired entries.
		MaxRetry:        3,                // Maximum number of retries for Redis operations.
	})

	if err != nil {
		// Panic if the Redis store cannot be created.
		panic(fmt.Sprintf("Failed to create Redis limiter store: %v", err))
	}

	// Create a rate limiter instance with the Redis store.
	instance := limiter.New(store, rate,
		limiter.WithTrustForwardHeader(true), // Trust the "X-Forwarded-For" header.
	)

	// Return the rate limiting middleware.
	return limitergin.NewMiddleware(instance,
		limitergin.WithKeyGetter(func(c *gin.Context) string {
			// Construct the rate limiting key using client IP and user ID (if available).
			key := c.ClientIP()

			if userID, exists := c.Get("userID"); exists {
				// Append user ID to the key if it exists.
				key += fmt.Sprintf(":user:%v", userID)
			}

			return key
		}),
		limitergin.WithErrorHandler(func(c *gin.Context, err error) {
			// Custom error handler for rate limiting errors.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": err.Error(),
			})
		}),
	)
}

// UserIDLimiter returns a middleware that performs rate limiting based on the user ID.
//
// The middleware will first check if the user ID exists in the context. If it does not,
// it will abort the request with a 401 Unauthorized status.
//
// If the user ID exists, it will get the user ID and use it as a key to store the
// rate limit context in memory. It will then check if the rate limit has been
// exceeded. If it has, it will abort the request with 429 Too Many Requests
// statuses and provide information about the rate limit in the response body.
//
// If the rate limit has not been exceeded, it will proceed to the next middleware
// or handler.
//
// The middleware takes a rate limiter.Rate as an argument, which specifies the
// rate limit to use.
func UserIDLimiter(rate limiter.Rate) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user ID exists in the context
		userID, exists := c.Get("userID")
		if !exists {
			// Abort the request if the user ID does not exist
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Create a memory store for the rate limit context
		store := memory.NewStore()

		// Use the user ID as the key for the rate limit context
		limiterKey := fmt.Sprintf("user:%v", userID)

		// Create a new rate limiter instance with the memory store and the given rate
		instance := limiter.New(store, rate, limiter.WithClientIPHeader("X-Forwarded-For"))

		// Get the rate limit context for the current request
		ctx, err := instance.Get(c, limiterKey)
		if err != nil {
			// Abort the request if there is an error getting the rate limit context
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		// Check if the rate limit has been exceeded
		if ctx.Reached {
			// Abort the request if the rate limit has been exceeded
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":  "Rate limit exceeded",
				"limit":  rate.Limit,
				"period": rate.Period.String(),
			})
			return
		}

		// Proceed to the next middleware or handler if the rate limit has not been exceeded
		c.Next()
	}
}

// IPWhitelist returns a middleware that allows requests only from a specified list of IP addresses.
//
// The middleware checks the client IP against the provided allowlist.
// If the IP is in the allowlist, the request proceeds to the next handler.
// Otherwise, it aborts the request with a 403 Forbidden status.
//
// Parameters:
//   - whitelist: A slice of strings containing the allowlisted IP addresses.
//
// Returns:
//   - gin.HandlerFunc: The Gin middleware function for IP allowlisting.
func IPWhitelist(whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the client IP address
		clientIP := c.ClientIP()

		// Iterate over the allowlist to check if the client IP is allowed
		for _, ip := range whitelist {
			if clientIP == ip {
				// Proceed to the next handler if the IP is allowlisted
				c.Next()
				return
			}
		}

		// Abort the request with a 403 Forbidden status if the IP is not allowed
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "IP not allowed"})
	}
}

// LoginRateLimiter returns a middleware that limits the number of login attempts
// from a given IP address.
//
// The middleware uses a Redis script to increment the login attempt counter
// for the client IP address.
//
// If the counter exceeds the configured limit (10 attempts in 1 second),
// the request is aborted with 429 Too Much Requests status.
//
// Otherwise, the request proceeds to the next handler.
//
// Parameters:
//   - None
//
// Returns:
//   - gin.HandlerFunc: The Gin middleware function for login rate limiting.
func LoginRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique request ID
		reqID := uuid.NewString()

		// Construct the Redis key for the login attempt counter
		key := fmt.Sprintf("rate_limit:login:%s", c.ClientIP())

		// Execute the Redis script to increment the counter and check if the limit has been exceeded
		allowed, err := resource.RedisClient.Eval(context.Background(), config.LuaScript, []string{key}, 1, 10).Int()
		if err != nil {
			// Log an error if the Redis eval fails
			resource.LoggerService.Error(fmt.Sprintf("Redis eval failed: %v", err))

			// Abort the request with a 500 Internal Server Error status
			resp.NewErrResp(c, http.StatusInternalServerError, fmt.Sprintf("Redis eval failed: %v", err), reqID)
			c.Abort()
			return
		}

		// Check if the rate limit has been exceeded
		if allowed == 0 {
			// Log an error if the rate limit has been exceeded
			resource.LoggerService.Error(fmt.Sprintf("Login rate limit exceeded"))

			// Abort the request with 429 Too Much Requests status
			resp.NewErrResp(c, http.StatusTooManyRequests, fmt.Sprintf("Too much login requests, rate limit exceeded"), reqID)
			c.Abort()
			return
		}

		// Proceed to the next handler if the rate limit has not been exceeded
		c.Next()
	}
}

// APIRateLimiter returns a middleware that limits the number of API requests
// from a given IP address.
//
// The middleware uses a Redis script to increment the API request counter
// for the client IP address.
//
// If the counter exceeds the configured limit (5 attempts in 1 second),
// the request is aborted with 429 Too Much Requests status.
//
// Otherwise, the request proceeds to the next handler.
//
// Parameters:
//   - None
//
// Returns:
//   - gin.HandlerFunc: The Gin middleware function for API rate limiting.
func APIRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique request ID
		reqID := uuid.NewString()

		// Construct the Redis key for the api attempt counter
		key := fmt.Sprintf("rate_limit:api:%s", c.ClientIP())

		// Execute the Redis script to increment the counter and check if the limit has been exceeded
		allowed, err := resource.RedisClient.Eval(context.Background(), config.LuaScript, []string{key}, 5, 1).Int()
		if err != nil {
			// Log an error if the Redis eval fails
			resource.LoggerService.Error(fmt.Sprintf("Redis eval failed: %v", err))

			// Abort the request with a 500 Internal Server Error status
			resp.NewErrResp(c, http.StatusInternalServerError, "Redis eval failed", reqID)
			c.Abort()
			return
		}

		// Check if the rate limit has been exceeded
		if allowed == 0 {
			// Log an error if the rate limit has been exceeded
			resource.LoggerService.Error(fmt.Sprintf("API request rate limit exceeded"))

			// Abort the request with 429 Too Much Requests status
			resp.NewErrResp(c, http.StatusTooManyRequests, "Too much API requests, rate limit exceeded", reqID)
			c.Abort()
			return
		}

		// Proceed to the next handler if the rate limit has not been exceeded
		c.Next()
	}
}
