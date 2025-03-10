package httpserver

import (
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	authcasbin "github.com/xiebingnote/go-gin-project/servers/httpserver/auth/casbin"
	"github.com/xiebingnote/go-gin-project/servers/httpserver/auth/jwt"

	"github.com/gin-gonic/gin"
)

// NewServer returns a new gin.Engine that can be used to start a web server.
//
// The returned gin.Engine is configured to serve the controllers registered in
// the controller package at "/web/api/xxx/v1". This URL path is used by the
// controllers to handle incoming requests.
//
// Users of NewServer should call the Run method on the returned gin.Engine to
// start the web server. The Run method will block until the web server is
// interrupted or an error occurs.
//
// The returned gin.Engine has been configured with the default gin.Logger and
// gin.Recovery middleware. The gin.Logger middleware logs each incoming
// request and response. The gin.Recovery middleware recovers from any panics
// that occur during the handling of a request and logs the panic.
func NewServer() *gin.Engine {
	// Set the gin mode to release.
	gin.SetMode(gin.ReleaseMode)

	// Create a new gin.Engine with the default middleware.
	// The default middleware consists of the gin.Logger and gin.Recovery
	// middleware.
	router := gin.Default()

	// JWT Register the auth endpoints.
	router.POST("/web/api/login", middleware.MemoryLimiter(config.LoginRate), jwt.Login)
	router.POST("/web/api/register", jwt.Register)

	// Register the controllers with the gin.Engine.
	// The controllers are registered at "/web/api/xxx/v1".
	api := router.Group("/web/api")
	// Apply JWT authentication middleware to the API group.
	api.Use(middleware.AuthMiddlewareJWT, middleware.APIRateLimiter())

	Router(api)

	// Return the configured gin.Engine.
	// Users of NewServer should call the Run method on the returned gin.Engine
	// to start the web server.
	return router
}

// NewServerCasbin returns a new gin.Engine configured with Casbin authorization.
//
// This function sets up default Casbin policies to manage access control for different roles.
// It creates a gin.Engine instance with default middleware (Logger and Recovery) and registers
// authentication endpoints for login and registration. The API routes are protected using
// Casbin middleware to enforce access control policies based on roles.
//
// The engine allows:
//   - The "admin" role to access all routes with any HTTP method.
//   - The "user" role to perform to GET requests on v1 endpoints, and any method on v1 endpoints.
//
// The function also groups the "alice" user under the "admin" role.
//
// Returns:
//
//	*gin.Engine: A gin.Engine instance configured for starting the web server with Casbin authorization.
func NewServerCasbin() *gin.Engine {
	// Add default Casbin policies for roles and permissions.
	// The "admin" role has access to all routes with any method.
	resource.Enforcer.AddPolicy("admin", "/*", "*")
	// The "user" role can perform GET requests on v1 endpoints.
	resource.Enforcer.AddPolicy("user", "/web/api/v1/*", "GET")
	// The "user" role can perform any method on v1 endpoints.
	resource.Enforcer.AddPolicy("user", "/web/api/v1/*", "*")
	// Group "alice" under the "admin" role group.
	resource.Enforcer.AddGroupingPolicy("alice", "admin")

	// Create a new gin.Engine with default middleware (Logger and Recovery).
	router := gin.Default()

	// Register the authentication endpoints for login and registration.
	router.POST("/web/api/v1/login", middleware.LoginRateLimiter(), authcasbin.Login)
	router.POST("/web/api/v1/register", authcasbin.Register)

	// Create an API group for versioned routes and apply Casbin authorization middleware.
	api := router.Group("/web/api")

	// Apply Casbin authorization middleware to the API group.
	api.Use(middleware.AuthMiddlewareCasbin(), middleware.APIRateLimiter())

	// Register additional routes with the API group.
	Router(api)

	// Return the configured gin.Engine for starting the web server.
	return router
}
