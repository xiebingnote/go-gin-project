package main

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/xiebingnote/go-gin-project/bootstrap"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/pkg/shutdown"
	"github.com/xiebingnote/go-gin-project/servers"

	"go.uber.org/zap"
)

const (
	// Server startup check timeout
	startupCheckTimeout = 100 * time.Millisecond
	// Server shutdown timeout
	serverShutdownTimeout = 5 * time.Second
	// Resource cleanup timeout
	resourceCleanupTimeout = 10 * time.Second
)

// The main is the entry point of the application, responsible for initializing
// components, starting the servers, and setting up a shutdown hook to handle
// graceful termination when receiving system signals.
// It performs the following tasks:
//  1. Create a background context for initialization tasks.
//  2. Initializes application components such as configuration, databases, and logging.
//  3. Starts the main and admin HTTP servers and captures any startup errors.
//  4. Monitor an error channel for any errors during server startup, logging a fatal error
//     if any occur, or confirming a successful startup.
//  5. Configure a shutdown hook to gracefully stop the servers and release resources,
//     logging any errors encountered during resource cleanup.
func main() {
	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error("Application panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			} else {
				// Fallback to standard log if logger is not available
				// Use log.Printf instead of debug.PrintStack() to avoid potential panic
				log.Printf("Application panic recovered: %v\n", r)
				log.Printf("Stack trace: %s\n", string(debug.Stack()))
			}
		}
	}()

	// Create a background context for initialization tasks
	ctx := context.Background()

	// Initialize application components such as configuration, databases, and logging
	bootstrap.MustInit(ctx)

	// Start the main and admin HTTP servers, capturing any startup errors
	mainSrv, adminSrv, errChan := servers.Start()

	// Monitor the error channel for any errors during server startup
	select {
	case err := <-errChan:
		// Log a fatal error and terminate if server startup fails
		resource.LoggerService.Fatal("❌ Server startup failed", zap.Error(err))
	case <-time.After(startupCheckTimeout):
		// Log successful server startup with their respective addresses
		resource.LoggerService.Info("✅ Servers started successfully",
			zap.String("main_server", mainSrv.Addr),
			zap.String("admin_server", adminSrv.Addr),
		)
	}

	// Configure a shutdown hook to cleanly stop servers and release resources
	shutdown.NewHook().Close(
		func() {
			// Shutdown the main server gracefully
			shutdownServer("main", mainSrv)
		},
		func() {
			// Shutdown the admin server gracefully
			shutdownServer("admin", adminSrv)
		},
		func() {
			// Perform cleanup of additional resources
			cleanupResources()
		},
	)
}

// shutdownServer closes the given HTTP server gracefully and logs the result.
// It creates a timeout context to ensure the server stops within a reasonable time.
//
// Parameters:
//   - name: The name of the server being shut down (e.g. "main" or "admin").
//   - srv: The HTTP server to be shut down.
func shutdownServer(name string, srv *http.Server) {
	// Use safe logging that checks if logger is available
	if resource.LoggerService != nil {
		resource.LoggerService.Info("Shutting down server", zap.String("server", name))
	} else {
		log.Printf("Shutting down server: %s", name)
	}

	// Create a context with timeout to ensure the server has time to stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	// Attempt to shut down the server gracefully
	if err := srv.Shutdown(shutdownCtx); err != nil {
		if resource.LoggerService != nil {
			resource.LoggerService.Error("❌ Server shutdown failed",
				zap.String("server", name),
				zap.Error(err),
			)
		} else {
			log.Printf("❌ Server shutdown failed (%s): %v", name, err)
		}
	} else {
		if resource.LoggerService != nil {
			resource.LoggerService.Info("Server stopped successfully", zap.String("server", name))
		} else {
			log.Printf("Server stopped successfully: %s", name)
		}
	}
}

// cleanupResources releases resources allocated by the application.
//
// The function creates a timeout context for the cleanup operation and
// attempts to close the resources allocated by the application. If the
// operation fails, it logs an error. If the operation succeeds, it logs a
// success message.
func cleanupResources() {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), resourceCleanupTimeout)
	defer cancel()

	if err := bootstrap.Close(cleanupCtx); err != nil {
		// Use standard log since Logger service may be closed
		log.Printf("❌ Resource cleanup failed: %v", err)
	} else {
		// Use standard log since Logger service is now closed
		log.Println("✅ Resource cleanup completed successfully")
	}
}
