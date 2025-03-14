package servers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/servers/httpserver"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Start initializes and starts both the main and admin HTTP servers.
//
// It creates a buffered error channel to capture any errors that occur
// when running the servers. The main server is started with the configuration
// and handler provided by newMainServer, while the admin server is started
// with the configuration and handler from newAdminServer. Both servers are
// run in separate goroutines, and any errors encountered are sent to the
// error channel.
//
// Parameters:
//   - ctx: The context for controlling server lifecycle and cancellation.
//
// Returns:
//   - mainSrv: The HTTP server for the main interface.
//   - adminSrv: The HTTP server for the admin interface.
//   - errChan: A channel for receiving errors from the servers.
func Start(_ context.Context) (mainSrv *http.Server, adminSrv *http.Server, errChan chan error) {
	// Create an error channel with a buffer size of 2 to capture errors from both servers.
	errChan = make(chan error, 2)

	// Start the main server with the provided configuration and handler.
	mainSrv = newMainServer(config.ServerConfig, httpserver.NewServer())
	go func() {
		// Run the main server and send any errors to the error channel.
		if err := runServer(mainSrv, "Main"); err != nil {
			errChan <- fmt.Errorf("main server failed: %w", err)
		}
	}()

	// Start the admin server with the provided configuration and handler.
	adminSrv = newAdminServer(config.ServerConfig, NewAdminServer())
	go func() {
		// Run the admin server and send any errors to the error channel.
		if err := runServer(adminSrv, "Admin"); err != nil {
			errChan <- fmt.Errorf("admin server failed: %w", err)
		}
	}()

	// Return the initialized servers and the error channel.
	return mainSrv, adminSrv, errChan
}

// newMainServer creates a new HTTP server for the main interface.
//
// The server is configured with the given configuration and uses the given
// handler for the main routes. This function creates a new HTTP server with the
// provided configuration and handler. The server will listen to the specified
// address and will use the provided handler for processing requests.
//
// Parameters:
//   - cfg: The ServerConfigEntry containing configuration settings for the main server.
//   - handler: The HTTP handler for processing main requests.
//
// Returns:
//   - A pointer to one http.Server configured for the main interface.
func newMainServer(cfg *config.ServerConfigEntry, handler http.Handler) *http.Server {
	// Create a new HTTP server with the given configuration.
	return &http.Server{
		Addr:         cfg.HTTPServer.Listen,                     // Listen to the address for the main server.
		Handler:      handler,                                   // HTTP handler for the main routes.
		ReadTimeout:  cfg.HTTPServer.ReadTimeout * time.Second,  // Read timeout for incoming requests.
		WriteTimeout: cfg.HTTPServer.WriteTimeout * time.Second, // Write timeout for outgoing responses.
		IdleTimeout:  cfg.HTTPServer.IdleTimeout * time.Second,  // Idle timeout for keep-alive connections.
	}
}

// newAdminServer creates and returns a new HTTP server for the admin interface.
//
// The server is configured using the provided ServerConfigEntry, which
// specifies the listen address and timeout settings. The provided handler
// is used to handle incoming requests on the admin routes.
//
// Parameters:
//   - cfg: The ServerConfigEntry containing configuration settings for the admin server.
//   - handler: The HTTP handler for processing admin requests.
//
// Returns:
//   - A pointer to one http.Server configured for the admin interface.
func newAdminServer(cfg *config.ServerConfigEntry, handler http.Handler) *http.Server {
	// Create a new HTTP server with the given configuration.
	return &http.Server{
		Addr:         cfg.AdminServer.Listen,                    // Listen to the address for the admin server.
		Handler:      handler,                                   // HTTP handler for the admin routes.
		ReadTimeout:  cfg.HTTPServer.ReadTimeout * time.Second,  // Read timeout for incoming requests.
		WriteTimeout: cfg.HTTPServer.WriteTimeout * time.Second, // Write timeout for outgoing responses.
		IdleTimeout:  cfg.HTTPServer.IdleTimeout * time.Second,  // Idle timeout for keep-alive connections.
	}
}

// runServer starts the HTTP server and listens for incoming requests.
//
// If the server fails to start or encounters an error (other than a closed server error),
// it returns an error with a formatted message indicating the server name.
//
// Parameters:
//   - srv: The HTTP server to run.
//   - name: The name of the server to a format in the error message.
//
// Returns:
//   - An error indicating the reason for the server failure.
func runServer(srv *http.Server, name string) error {
	// Attempt to start the server and listen for incoming requests.
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		// Return a formatted error message if the server fails to start.
		resource.LoggerService.Error(fmt.Sprintf("%s server failed: %v", name, err))
		return fmt.Errorf("%s server failed: %w", name, err)
	}
	// Return nil if the server shuts down gracefully.
	return nil
}

// NewAdminServer returns a new HTTP handler for the admin interface.
//
// The returned handler registers the following endpoints:
//   - /debug/pprof/ (via gin.WrapH(http.DefaultServeMux)): the pprof debug endpoints.
//   - /metrics: a test endpoint that returns a 200 OK response with the text "Metrics endpoint".
//   - /test: a test endpoint that returns a 200 OK response with a UUID.
//
// The handler also uses the Gin recovery middleware to recover from panics and return a 500 Internal Server Error response.
// The middleware.PrometheusMiddleware is used to register the Prometheus metrics endpoint.
func NewAdminServer() http.Handler {
	// Create a new Gin router for handling admin routes.
	router := gin.New()
	// Use the Gin recovery middleware to recover from panics and return a 500 Internal Server Error response.
	router.Use(gin.Recovery(), middleware.PrometheusMiddleware())

	// Register the pprof debug endpoints using the default HTTP ServeMux.
	// The pprof package provides the http.DefaultServeMux handler, which serves the pprof debug endpoints.
	router.GET("/debug/pprof/", gin.WrapH(http.DefaultServeMux))
	// The pprof package also provides a handler for the ":profile" endpoint.
	// This endpoint serves the pprof profile data for the given profile name.
	router.GET("/debug/pprof/:profile", gin.WrapH(http.DefaultServeMux))

	// Register the Prometheus metrics endpoint.
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Register a test endpoint that returns a 200 OK response with a UUID.
	// This endpoint can be used to test the admin server.
	router.GET("/test", func(c *gin.Context) {
		// Respond with a 200-OK status and a message.
		resp.NewOKResp(c, "Metrics endpoint test", uuid.NewString())
	})

	// Return the configured Gin router as the admin HTTP handler.
	return router
}
