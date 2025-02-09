package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"project/bootstrap"
	"project/library/resource"
	"project/pkg/shutdown"
	"project/servers"

	"go.uber.org/zap"
)

// main is the entry point of the application, responsible for initializing
// components, starting the servers, and setting up a shutdown hook to handle
// graceful termination when receiving system signals.
// It performs the following tasks:
//  1. Creates a background context for initialization tasks.
//  2. Initializes application components such as configuration, databases, and logging.
//  3. Starts the main and admin HTTP servers and captures any startup errors.
//  4. Monitors an error channel for any errors during server startup, logging a fatal error
//     if any occur, or confirming successful startup.
//  5. Configures a shutdown hook to gracefully stop the servers and release resources,
//     logging any errors encountered during resource cleanup.
func main() {
	// Create a background context for initialization tasks
	ctx := context.Background()

	// Initialize application components such as configuration, databases, and logging
	bootstrap.MustInit(ctx)

	// Start the main and admin HTTP servers, capturing any startup errors
	mainSrv, adminSrv, errChan := servers.Start(ctx)

	// Monitor the error channel for any errors during server startup
	select {
	case err := <-errChan:
		// Log a fatal error and terminate if server startup fails
		resource.LoggerService.Fatal("Server startup failed", zap.Error(err))
	case <-time.After(100 * time.Millisecond): // Short delay to check for immediate errors
		// Log successful server startup with their respective addresses
		resource.LoggerService.Info("Servers started successfully",
			zap.String("main_addr", mainSrv.Addr),
			zap.String("admin_addr", adminSrv.Addr))
	}

	// Configure a shutdown hook to cleanly stop servers and release resources
	shutdown.NewHook().Close(
		func() {
			// Shutdown the main server gracefully
			shutdownServer("Main", mainSrv)
		},
		func() {
			// Shutdown the admin server gracefully
			shutdownServer("Admin", adminSrv)
		},
		func() {
			// Perform cleanup of additional resources
			if err := bootstrap.Close(); err != nil {
				// Log any errors encountered during resource cleanup
				resource.LoggerService.Error("Resource cleanup failed", zap.Error(err))
			}
		},
	)
}

// shutdownServer closes the given HTTP server and logs any errors that occur.
// It will block until the server has stopped or the given context is canceled.
// If the context is canceled before the server has stopped, the server will be
// interrupted and an error will be logged.
//
// Parameters:
//   - name: The name of the server being shut down (e.g. "Main" or "Admin").
//   - srv: The HTTP server to be shut down.
func shutdownServer(name string, srv *http.Server) {
	resource.LoggerService.Info(fmt.Sprintf("Closing %s server...", name))
	// Create a context with a 5 second timeout to ensure the server has time to stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to shut down the server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// If there is an error, log the error and the server name
		resource.LoggerService.Error(fmt.Sprintf("%s server shutdown error", name), zap.Error(err))
	} else {
		// Log the successful shutdown of the server
		resource.LoggerService.Info(fmt.Sprintf("Stopped %s server", name))
	}
}
