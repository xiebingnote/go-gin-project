package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	authcasbin "github.com/xiebingnote/go-gin-project/servers/httpserver/auth/casbin"
	"github.com/xiebingnote/go-gin-project/servers/httpserver/auth/jwt"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"go.uber.org/zap"
)

// ServerOptions represents the configuration options for the HTTP server.
type ServerOptions = config.ServerOptions

// ServerRateLimitConfig represents the configuration options for rate limiting
type ServerRateLimitConfig = config.ServerRateLimitConfig

// DefaultServerOptions returns the default server options
//
// This function returns a copy of the default server options. You can use this
// function to create a new server with the default options.
//
// Returns:
//   - *ServerOptions:  a pointer to the default server options
func DefaultServerOptions() *ServerOptions {
	if config.ServerConfig == nil {
		panic("server configuration is not initialized")
	}
	opts := config.ServerConfig.Options
	opts.TrustedProxies = append([]string(nil), opts.TrustedProxies...)
	opts.CORSAllowedOrigins = append([]string(nil), opts.CORSAllowedOrigins...)
	if opts.RateLimitConfig != nil {
		limits := *opts.RateLimitConfig
		opts.RateLimitConfig = &limits
	}
	return &opts
}

// NewServer creates a new HTTP server instance
//
// This function creates a new HTTP server with the default configuration.
// The server is configured with JWT authentication, rate limiting, circuit
// breaking, and monitoring. If you need to customize the configuration, you can use
// NewServerWithOptions instead.
//
// Returns:
//   - *gin.Engine:  the configured Gin engine instance
func NewServer() *gin.Engine {
	return NewServerWithOptions(DefaultServerOptions())
}

// NewServerWithOptions creates an HTTP server using custom configuration.
//
// This function configures the Gin engine based on the provided options, including
// middleware, routes, authentication, etc. It provides full server configuration
// capabilities and supports various custom options.
//
// Parameters:
//   - opts: Server configuration options
//
// Returns:
//   - *gin.Engine: Configured Gin engine instance
func NewServerWithOptions(opts *ServerOptions) *gin.Engine {
	// Validate the configuration
	if err := Validate(opts); err != nil {
		if resource.LoggerService != nil {
			resource.LoggerService.Error("Invalid server options", zap.Error(err))
		}
		panic(fmt.Sprintf("Invalid server options: %v", err))
	}

	if opts.EnableAuth {
		if err := middleware.LoadJWTSecretFromEnv(); err != nil {
			panic(err)
		}
		if opts.AuthType == "casbin" && resource.Enforcer == nil {
			panic("Casbin enforcer must be initialized before serving authenticated routes")
		}
	}

	// Set Gin mode
	gin.SetMode(opts.Mode)

	// Create a new Gin engine without default middleware
	router := gin.New()

	// Configure trusted proxies
	if len(opts.TrustedProxies) > 0 {
		if err := router.SetTrustedProxies(opts.TrustedProxies); err != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error("Failed to set trusted proxies", zap.Error(err))
			}
		}
	}

	// Add base middleware
	setupBaseMiddleware(router)

	// Add security middleware if enabled
	if opts.EnableSecurity {
		setupSecurityMiddleware(router, opts)
	}

	// Add monitoring middleware if metrics are enabled
	if opts.EnableMetrics {
		router.Use(middleware.PrometheusMiddleware())
	}

	// Set up authentication routes
	setupAuthRoutes(router, opts)

	// Set up API route group with middleware
	api := router.Group("/web/api")
	setupAPIMiddleware(api, opts)

	// Register business routes
	Router(api)

	// Log server configuration if logging service is available
	if resource.LoggerService != nil {
		resource.LoggerService.Info("✅ HTTP server configured successfully",
			zap.String("mode", opts.Mode),
			zap.String("auth_type", opts.AuthType),
			zap.Bool("enable_metrics", opts.EnableMetrics),
			zap.Bool("enable_cors", opts.EnableCORS),
		)
	}

	return router
}

// Validate checks the validity of the server configuration options.
//
// This function ensures that the provided server options are valid. It checks
// the mode, authentication type, and timeout values to ensure they are within
// acceptable ranges.
//
// Parameters:
//   - opts: *ServerOptions, the server configuration options to be validated.
//
// Returns:
//   - error: An error if any of the options are invalid, otherwise nil.
func Validate(opts *ServerOptions) error {
	if opts == nil {
		return fmt.Errorf("server options are required")
	}
	if opts.EnableCORS {
		if err := middleware.ValidateCORSOrigins(opts.CORSAllowedOrigins); err != nil {
			return err
		}
	}
	if limits := opts.RateLimitConfig; limits != nil && (limits.LoginLimit < 0 || limits.APILimit < 0 || limits.PublicLimit < 0) {
		return fmt.Errorf("rate limits must not be negative")
	}

	// Check if the mode is valid
	if opts.Mode != gin.DebugMode && opts.Mode != gin.ReleaseMode && opts.Mode != gin.TestMode {
		return fmt.Errorf("invalid gin mode: %s", opts.Mode)
	}

	// Check if the authentication type is valid
	if opts.AuthType != "jwt" && opts.AuthType != "casbin" {
		return fmt.Errorf("invalid auth type: %s", opts.AuthType)
	}

	// Ensure read timeout is a positive value
	if opts.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}

	// Ensure write timeout is a positive value
	if opts.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}

	// Ensure shutdown timeout is a positive value
	if opts.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	return nil
}

// setupBaseMiddleware sets up the base middleware for the gin server.
//
// The base middleware includes a custom logger, a recovery middleware, and a
// request ID middleware.
func setupBaseMiddleware(router *gin.Engine) {
	// Custom logger middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	}))

	// Recovery middleware
	//
	// This middleware recovers from panic and logs the error. It also returns
	// a JSON response with a 500 status code.
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if resource.LoggerService != nil {
			resource.LoggerService.Error("Panic recovered",
				zap.Any("error", recovered),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Server Error",
			"code":  "INTERNAL_ERROR",
		})
	}))

	// Request ID middleware
	//
	// This middleware sets a request ID for each request.
	router.Use(middleware.RequestIDMiddleware())
}

// setupSecurityMiddleware sets up the security middleware for the router.
//
// It enables CORS, sets up security headers, and enables rate limiting if
// configured.
func setupSecurityMiddleware(router *gin.Engine, opts *ServerOptions) {
	// Enable CORS if configured
	if opts.EnableCORS {
		router.Use(middleware.CORSMiddleware(opts.CORSAllowedOrigins, opts.CORSAllowCredentials))
	}

	// Set up security headers for all requests
	router.Use(middleware.SecurityHeadersMiddleware())

	// Enable rate limiting if configured
	if opts.RateLimitConfig != nil {
		// Use Redis for rate limiting if configured and Redis is available
		if opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
			router.Use(middleware.RedisLimiter(publicRate(opts)))
		} else if opts.RateLimitConfig.EnableMemory || opts.RateLimitConfig.EnableRedis {
			// Use in-memory rate limiting if Redis is not available
			router.Use(middleware.MemoryLimiter(publicRate(opts)))
		}
	}
}

// setupAuthRoutes sets up the authentication routes based on the authentication type specified in the options.
//
// The function configures routes for login and registration endpoints using either JWT or Casbin authentication.
// It also applies rate limiting based on the configuration provided in ServerOptions.
func setupAuthRoutes(router *gin.Engine, opts *ServerOptions) {
	if !opts.EnableAuth {
		return
	}
	loginLimiter := middleware.MemoryLimiter(loginRate(opts))
	if opts.RateLimitConfig != nil && opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
		loginLimiter = middleware.LoginRateLimiter(loginRate(opts))
	}
	switch opts.AuthType {
	case "jwt":
		router.POST("/web/api/login", loginLimiter, jwt.Login)
		router.POST("/web/api/register", jwt.Register)
	case "casbin":
		router.POST("/web/api/v1/login", loginLimiter, authcasbin.Login)
		router.POST("/web/api/v1/register", authcasbin.Register)
	}
}

// setupAPIMiddleware sets up the middleware for the API routes.
//
// This function configures authentication, rate limiting, and circuit breaking
// before business routes are registered. Each server owns one breaker manager.
//
// Parameters:
//   - api: The API route group to which middleware is applied.
//   - opts: The server configuration options.
func setupAPIMiddleware(api *gin.RouterGroup, opts *ServerOptions) {
	// Apply authentication middleware based on the configured authentication type
	if opts.EnableAuth {
		switch opts.AuthType {
		case "jwt":
			api.Use(middleware.AuthMiddlewareJWT)
		case "casbin":
			api.Use(middleware.AuthMiddlewareCasbin(), middleware.CasbinMiddleware(resource.Enforcer))
		}
	}

	// Apply rate limiting middleware if configured
	if opts.RateLimitConfig != nil {
		if opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
			// Use Redis-based rate limiting
			api.Use(middleware.APIRateLimiter(apiRate(opts)))
		} else if opts.RateLimitConfig.EnableMemory || opts.RateLimitConfig.EnableRedis {
			// Use in-memory rate limiting
			api.Use(middleware.MemoryLimiter(apiRate(opts)))
		}
	}

	// Reuse one manager across this server's business routes. Authentication and
	// rate-limit rejections must finish before requests enter circuit statistics.
	manager := middleware.NewCircuitBreakerManager(resource.LoggerService)
	api.Use(middleware.CustomCircuitBreakerMiddleware(manager, middleware.DefaultCircuitBreakerConfig))
}

// Limit overrides use requests per minute, matching conf/server.toml.
func loginRate(opts *ServerOptions) limiter.Rate {
	if cfg := opts.RateLimitConfig; cfg != nil && cfg.LoginLimit > 0 {
		return limiter.Rate{Period: time.Minute, Limit: int64(cfg.LoginLimit)}
	}
	return config.LoginRate
}
func apiRate(opts *ServerOptions) limiter.Rate {
	if cfg := opts.RateLimitConfig; cfg != nil && cfg.APILimit > 0 {
		return limiter.Rate{Period: time.Minute, Limit: int64(cfg.APILimit)}
	}
	return config.APIRate
}
func publicRate(opts *ServerOptions) limiter.Rate {
	if cfg := opts.RateLimitConfig; cfg != nil && cfg.PublicLimit > 0 {
		return limiter.Rate{Period: time.Minute, Limit: int64(cfg.PublicLimit)}
	}
	return config.PublicRate
}

// NewServerCasbin creates an HTTP server with Casbin authorization enabled.
//
// Casbin policies must be provisioned before serving traffic. No sample policies
// or privileged role assignments are added automatically.
//
// Returns:
//   - *gin.Engine: A configured Gin engine instance with Casbin authorization enabled.
func NewServerCasbin() *gin.Engine {
	opts := DefaultServerOptions()
	opts.AuthType = "casbin"
	return NewServerWithOptions(opts)
}

// NewServerJWT creates an HTTP server with JWT authentication enabled.
//
// This function creates a Gin engine instance with JWT authentication enabled.
// JWT (JSON Web Token) is a token-based authentication mechanism that is
// stateless and does not require a session.
//
// Returns:
//   - *gin.Engine: A Gin engine instance with JWT authentication enabled.
func NewServerJWT() *gin.Engine {
	opts := DefaultServerOptions()
	opts.AuthType = "jwt"
	return NewServerWithOptions(opts)
}

// NewServerWithoutAuth creates an HTTP server without authentication.
// ¬
// This function is suitable for public APIs or internal services that do not
// require user authentication. It still includes other security measures such
// as rate limiting and monitoring middleware.
//
// Returns:
//   - *gin.Engine: A Gin engine instance without authentication enabled.
func NewServerWithoutAuth() *gin.Engine {
	// Retrieve the default server options
	opts := DefaultServerOptions()

	// Disable authentication for this server instance
	opts.EnableAuth = false

	// Create and return a new server with the specified options
	return NewServerWithOptions(opts)
}
