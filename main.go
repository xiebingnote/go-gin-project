package main

import (
	"context"
	"log"
	"project/bootstrap"
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
	srv, errChan := servers.Start(ctx)

	// Wait for any errors while starting the server
	select {
	case err := <-errChan:
		// Log a fatal error if the server fails to start
		log.Fatalf("Failed to start HTTP server: %v", err)
	default:
		log.Printf("✅ HTTP server listening on %s", srv.Addr)
	}

	// Create a shutdown hook to handle graceful shutdown
	shutdown.NewHook().Close(
		func() {
			log.Println("🛑 Shutting down HTTP server...")
			// Create a context with timeout for server shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Shutdown the server gracefully
			if err := srv.Shutdown(shutdownCtx); err != nil {
				// Log any errors during server shutdown
				log.Printf("HTTP server shutdown error: %v", err)
			} else {
				log.Println("✅ HTTP server stopped gracefully")
			}
		},
		func() {
			// Close application resources
			if err := bootstrap.Close(); err != nil {
				// Log any errors during resource cleanup
				log.Printf("Resource cleanup error: %v", err)
			} else {
				log.Println("✅ All resources cleaned up")
			}
		},
	)
}
