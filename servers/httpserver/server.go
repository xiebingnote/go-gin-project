package httpserver

import (
	"fmt"
	"net/http"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	authcasbin "github.com/xiebingnote/go-gin-project/servers/httpserver/auth/casbin"
	"github.com/xiebingnote/go-gin-project/servers/httpserver/auth/jwt"

	"github.com/gin-gonic/gin"
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
	//return config.DefaultServerOptions()
	return &ServerOptions{
		Mode:            config.ServerConfig.Options.Mode,
		EnablePprof:     config.ServerConfig.Options.EnablePprof,
		EnableMetrics:   config.ServerConfig.Options.EnableMetrics,
		TrustedProxies:  config.ServerConfig.Options.TrustedProxies,
		EnableCORS:      config.ServerConfig.Options.EnableCORS,
		EnableSecurity:  config.ServerConfig.Options.EnableSecurity,
		AuthType:        config.ServerConfig.Options.AuthType,
		EnableAuth:      config.ServerConfig.Options.EnableAuth,
		ReadTimeout:     config.ServerConfig.Options.ReadTimeout,
		WriteTimeout:    config.ServerConfig.Options.WriteTimeout,
		IdleTimeout:     config.ServerConfig.Options.IdleTimeout,
		ShutdownTimeout: config.ServerConfig.Options.ShutdownTimeout,
		RateLimitConfig: &ServerRateLimitConfig{
			EnableRedis:  config.ServerConfig.Options.RateLimitConfig.EnableRedis,
			EnableMemory: config.ServerConfig.Options.RateLimitConfig.EnableMemory,
			LoginLimit:   config.ServerConfig.Options.RateLimitConfig.LoginLimit,
			APILimit:     config.ServerConfig.Options.RateLimitConfig.APILimit,
			PublicLimit:  config.ServerConfig.Options.RateLimitConfig.PublicLimit,
		},
	}
}

// NewServer creates a new HTTP server instance
//
// This function creates a new HTTP server with the default configuration.
// The server is configured with JWT authentication, rate limiting, and
// monitoring. If you need to customize the configuration, you can use
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
		router.Use(middleware.CORSMiddleware())
	}

	// Set up security headers for all requests
	router.Use(middleware.SecurityHeadersMiddleware())

	// Enable rate limiting if configured
	if opts.RateLimitConfig != nil {
		// Use Redis for rate limiting if configured and Redis is available
		if opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
			router.Use(middleware.RedisLimiter(config.PublicRate))
		} else if opts.RateLimitConfig.EnableMemory {
			// Use in-memory rate limiting if Redis is not available
			router.Use(middleware.MemoryLimiter(config.PublicRate))
		}
	}
}

// setupAuthRoutes sets up the authentication routes based on the authentication type specified in the options.
//
// The function configures routes for login and registration endpoints using either JWT or Casbin authentication.
// It also applies rate limiting based on the configuration provided in ServerOptions.
func setupAuthRoutes(router *gin.Engine, opts *ServerOptions) {
	// Return early if authentication is not enabled
	if !opts.EnableAuth {
		return
	}

	// Determine the authentication type and set up routes accordingly
	switch opts.AuthType {
	case "jwt":
		// Set up JWT authentication routes
		// Apply in-memory rate limiting by default
		loginLimiter := middleware.MemoryLimiter(config.LoginRate)

		// Use Redis for rate limiting if configured and Redis is available
		if opts.RateLimitConfig != nil && opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
			loginLimiter = middleware.LoginRateLimiter()
		}

		// Register login and register routes using JWT handlers
		router.POST("/web/api/login", loginLimiter, jwt.Login)
		router.POST("/web/api/register", jwt.Register)

	case "casbin":
		// Set up Casbin authentication routes
		setupCasbinPolicies()

		// Register login and register routes using Casbin handlers with rate limiting
		router.POST("/web/api/v1/login", middleware.LoginRateLimiter(), authcasbin.Login)
		router.POST("/web/api/v1/register", authcasbin.Register)
	}
}

// setupAPIMiddleware sets up the middleware for the API routes.
//
// This function configures authentication and rate limiting middleware
// for the API route group based on the server options provided.
//
// Parameters:
//   - api: The API route group to which middleware is applied.
//   - opts: The server configuration options.
func setupAPIMiddleware(api *gin.RouterGroup, opts *ServerOptions) {
	// Return early if authentication is not enabled
	if !opts.EnableAuth {
		return
	}

	// Apply authentication middleware based on the configured authentication type
	switch opts.AuthType {
	case "jwt":
		// Use JWT authentication middleware
		api.Use(middleware.AuthMiddlewareJWT)
	case "casbin":
		// Use Casbin authentication middleware
		api.Use(middleware.AuthMiddlewareCasbin())
	}

	// Apply rate limiting middleware if configured
	if opts.RateLimitConfig != nil {
		if opts.RateLimitConfig.EnableRedis && resource.RedisClient != nil {
			// Use Redis-based rate limiting
			api.Use(middleware.APIRateLimiter())
		} else if opts.RateLimitConfig.EnableMemory {
			// Use in-memory rate limiting
			api.Use(middleware.MemoryLimiter(config.PublicRate))
		}
	}
}

// setupCasbinPolicies sets up the default Casbin policies and grouping policies.
//
// This function configures the default access control policies for different roles
// and adds specific users to role groups. If the Casbin enforcer is not initialized,
// the function exits early.
func setupCasbinPolicies() {
	// Return early if the Casbin enforcer is not initialized
	if resource.Enforcer == nil {
		return
	}

	// Define default policies for role-based access control
	policies := [][]string{
		// "admin" role has access to all routes and HTTP methods
		{"admin", "/*", "*"},
		// "user" role can perform GET requests on v1 endpoints
		{"user", "/web/api/v1/*", "GET"},
		// "user" role can perform any method on v1 endpoints
		{"user", "/web/api/v1/*", "*"},
	}

	// Add each policy to the Casbin enforcer
	for _, policy := range policies {
		if _, err := resource.Enforcer.AddPolicy(policy); err != nil {
			// Log an error if adding the policy fails
			if resource.LoggerService != nil {
				resource.LoggerService.Error("Failed to add Casbin policy",
					zap.Strings("policy", policy),
					zap.Error(err),
				)
			}
		}
	}

	// Add grouping policy to associate users with roles
	if _, err := resource.Enforcer.AddGroupingPolicy("alice", "admin"); err != nil {
		// Log an error if adding the grouping policy fails
		if resource.LoggerService != nil {
			resource.LoggerService.Error("Failed to add Casbin grouping policy", zap.Error(err))
		}
	}
}

// NewServerCasbin creates an HTTP server with Casbin authorization enabled.
//
// Casbin is used for role-based access control (RBAC). The default policies
// are set up to manage access control for different roles. The function also
// creates a configured Gin engine instance.
//
// Default policies:
//   - "admin" role has access to all routes and HTTP methods
//   - "user" role can perform GET requests on v1 endpoints and any method
//   - "alice" user is grouped into the "admin" role
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
