package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"project/bootstrap"
	"project/library/resource"
	"project/pkg/shutdown"
	"project/servers"
	"time"
)

// main starts the entire application, initializes the necessary components, and starts the server.
// It also sets up a shutdown hook to listen for the operating system's SIGINT and SIGTERM signals.
// When a signal is received, the hook will execute the functions passed to the Close method in sequence.
func main() {
	// Create a context to be used for initialization
	ctx := context.Background()

	// Initialize necessary components of the application
	bootstrap.MustInit(ctx)

	// Start the HTTP server and receive an error channel
	mainSrv, adminSrv, errChan := servers.Start(ctx)

	// Wait for any errors while starting the server
	select {
	case err := <-errChan:
		// Log and exit if there is an error starting the servers
		resource.LoggerService.Fatal("Server startup failed", zap.Error(err))
	case <-time.After(100 * time.Millisecond): // Wait briefly to ensure no immediate errors
		// Log the successful startup of the servers
		resource.LoggerService.Info("Servers started successfully",
			zap.String("main_addr", mainSrv.Addr),
			zap.String("admin_addr", adminSrv.Addr))
	}

	// Set up a shutdown hook to cleanly shut down the servers and resources
	shutdown.NewHook().Close(
		func() {
			// Shutdown the main server
			shutdownServer("Main", mainSrv)
		},
		func() {
			// Shutdown the admin server
			shutdownServer("Admin", adminSrv)
		},
		func() {
			// Cleanup any other resources
			if err := bootstrap.Close(); err != nil {
				resource.LoggerService.Error("Resource cleanup failed", zap.Error(err))
			}
		},
	)
}

// shutdownServer closes the given HTTP server and logs any errors that occur.
// It will block until the server has stopped or the given context is canceled.
// If the context is canceled before the server has stopped, the server will be
// interrupted and an error will be logged.
func shutdownServer(name string, srv *http.Server) {
	resource.LoggerService.Info(fmt.Sprintf("Closing %s server...", name))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("%s server shutdown error", name), zap.Error(err))
	} else {
		resource.LoggerService.Info(fmt.Sprintf("Stopped %s server", name))
	}
}
