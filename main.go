package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/xiebingnote/go-gin-project/bootstrap"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/pkg/shutdown"
	"github.com/xiebingnote/go-gin-project/servers"

	"go.uber.org/zap"
)

// AppTimeouts represents the timeouts used during startup and shutdown.
type AppTimeouts struct {
	StartupCheck    time.Duration
	ServerShutdown  time.Duration
	ResourceCleanup time.Duration
}

// AppTimeouts represents the timeouts used during startup and shutdown.
var defaultTimeouts = AppTimeouts{
	StartupCheck:    5 * time.Second,
	ServerShutdown:  10 * time.Second,
	ResourceCleanup: 15 * time.Second,
}

// StartupError represents an error that occurred during startup.
type StartupError struct {
	Component string
	Err       error
	Retryable bool
}

// ServerPair represents a pair of main and admin servers.
type ServerPair struct {
	Main  *http.Server
	Admin *http.Server
}

// Error implements the error interface and returns a string representation
// of the error.
//
// The format of the string is: "startup failed for <component>: <err>".
//
// The component is the name of the component that failed to start, and
// err is the underlying error that caused the startup to fail.
func (e *StartupError) Error() string {
	return fmt.Sprintf("startup failed for %s: %v", e.Component, e.Err)
}

// Unwrap returns the underlying error that caused the startup to fail.
// It implements the `Unwrap` method of the `errors.Unwrap` interface.
func (e *StartupError) Unwrap() error {
	return e.Err
}

// main is the entry point of the application, responsible for initializing
// components, starting the servers, starting background tasks, logging startup
// metrics, and setting up graceful shutdown to handle termination signals.
//
// The main function performs the following tasks:
//  1. Initializes all components using the bootstrap package.
//  2. Starts the main and admin servers and records startup metrics.
//  3. Starts background tasks such as memory monitoring and uptime updates.
//  4. Logs the time taken to complete startup and the addresses of the main and
//     admin servers.
//  5. Sets up graceful shutdown to handle termination signals.
func main() {
	// Record the application start time for metrics
	startTime := time.Now()
	middleware.AppStartTime.WithLabelValues("1.0.0").Set(float64(startTime.Unix()))

	// Handle panics gracefully and log them
	defer func() {
		if r := recover(); r != nil {
			handlePanic(r)
		}
	}()

	// 1. Initialize all components
	ctx := context.Background()
	bootstrap.MustInit(ctx)

	// 2. Start the servers and monitor startup metrics
	serverPair, err := startServersWithMetrics()
	if err != nil {
		resource.LoggerService.Fatal("Failed to start serverPair", zap.Error(err))
	}

	// 3. Start background tasks such as memory monitoring and uptime updates
	startBackgroundTasks()

	// 4. Log the time taken to complete startup
	startupDuration := time.Since(startTime)
	middleware.ServerStartupDuration.Observe(startupDuration.Seconds())
	resource.LoggerService.Info("✅ Application started successfully",
		zap.Duration("startup_duration", startupDuration),
		zap.String("main_server", serverPair.Main.Addr),
		zap.String("admin_server", serverPair.Admin.Addr),
	)

	// 5. Set up graceful shutdown to handle termination signals
	setupGracefulShutdown(serverPair, defaultTimeouts)
}

// startServersWithMetrics starts the main and admin servers and records startup metrics.
// It returns a ServerPair containing the main and admin servers, or an error if the startup fails.
func startServersWithMetrics() (*ServerPair, error) {
	// Record the time taken to start the servers
	defer middleware.Timer.ObserveDuration()

	// Start the main and admin servers
	mainSrv, adminSrv, errChan := servers.Start()

	// Wait for the servers to start or return an error if the startup fails
	select {
	case err := <-errChan:
		// If the startup fails, return a StartupError with the component name and error
		return nil, &StartupError{
			Component: "servers",
			Err:       err,
			Retryable: false,
		}
	case <-time.After(100 * time.Millisecond):
		// If the servers start successfully, return a ServerPair containing the main and admin servers
	}

	return &ServerPair{
		Main:  mainSrv,
		Admin: adminSrv,
	}, nil
}

// startBackgroundTasks starts the background tasks.
//
// This function starts the background tasks such as memory monitoring
// and uptime updating.
func startBackgroundTasks() {
	// Start memory monitoring
	//
	// The memory monitor logs the current memory usage every 5
	// minutes and triggers a manual GC if the memory usage exceeds
	// 500MB.
	startMemoryMonitor()

	// Start updating the uptime
	//
	//  updater logs the current uptime every 30 seconds.
	startUptimeUpdater()
}

// startMemoryMonitor starts monitoring memory usage and logs memory statistics
// every 5 minutes. If the allocated memory exceeds 500MB, it triggers a manual
// garbage collection (GC) to free up memory.
func startMemoryMonitor() {
	// Create a ticker that ticks every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)

	// Run the monitoring logic in a separate goroutine
	go func() {
		defer ticker.Stop() // Ensure ticker is stopped when the goroutine exits

		// Continuously monitor memory usage at each tick
		for range ticker.C {
			var m runtime.MemStats
			// Read current memory statistics
			runtime.ReadMemStats(&m)

			// Log memory statistics
			resource.LoggerService.Info("Memory stats",
				zap.Uint64("alloc_mb", m.Alloc/1024/1024), // Allocated memory in MB
				zap.Uint64("sys_mb", m.Sys/1024/1024),     // Total system memory in MB
				zap.Uint32("num_gc", m.NumGC),             // Number of completed GC cycles
				zap.Uint64("heap_objects", m.HeapObjects), // Number of allocated heap objects
			)

			// Trigger GC if allocated memory exceeds 1GB
			if m.Alloc > 1024*1024*1024 {
				runtime.GC()
				resource.LoggerService.Info("Triggered manual GC due to high memory usage")
			}
		}
	}()
}

// startUptimeUpdater starts the uptime updater goroutine.
//
// The uptime updater logs the current uptime every 30 seconds.
func startUptimeUpdater() {
	startTime := time.Now()
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop() // Ensure ticker is stopped when the goroutine exits

		// Continuously update the uptime every 30 seconds
		for range ticker.C {
			uptime := time.Since(startTime).Seconds()
			middleware.AppUptime.WithLabelValues("1.0.0").Set(uptime)
		}
	}()
}

// setupGracefulShutdown sets up the shutdown hook to handle termination signals.
//
// The shutdown hook is used to perform the following tasks in order:
//  1. Shut down the main server with a timeout.
//  2. Shut down the admin server with a timeout.
//  3. Clean up resources with a timeout.
func setupGracefulShutdown(servers *ServerPair, timeouts AppTimeouts) {
	shutdown.NewHook().Close(
		// Close the main server with a timeout
		func() {
			shutdownServerWithTimeout("main", servers.Main, timeouts.ServerShutdown)
		},
		// Close the admin server with a timeout
		func() {
			shutdownServerWithTimeout("admin", servers.Admin, timeouts.ServerShutdown)
		},
		// Clean up resources with a timeout
		func() {
			cleanupResourcesWithTimeout(timeouts.ResourceCleanup)
		},
	)
}

// shutdownServerWithTimeout gracefully shuts down the specified HTTP server
// within a given timeout period. It logs the shutdown process and any errors encountered.
//
// Parameters:
//   - name: The name of the server being shut down.
//   - srv: The HTTP server instance to be shut down.
//   - timeout: The maximum duration allowed for the server to shut down.
func shutdownServerWithTimeout(name string, srv *http.Server, timeout time.Duration) {
	// Log that the server shutdown process has started
	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 Shutting down server",
			zap.String("server", name),
			zap.Duration("timeout", timeout),
		)
	}

	// Create a context with the specified timeout to control the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt to shut down the server gracefully
	if err := srv.Shutdown(ctx); err != nil {
		// Log an error if the server fails to shut down
		if resource.LoggerService != nil {
			resource.LoggerService.Error("Server shutdown failed",
				zap.String("server", name),
				zap.Error(err),
			)
		} else {
			// Fallback to standard log if logger is not available
			log.Printf("❌ Server shutdown failed (%s): %v", name, err)
		}
	} else {
		// Log a success message if the server stops successfully
		if resource.LoggerService != nil {
			resource.LoggerService.Info("✅ Server stopped successfully",
				zap.String("server", name))
		} else {
			// Fallback to standard log if logger is not available
			log.Printf("✅ Server stopped successfully: %s", name)
		}
	}
}

// cleanupResourcesWithTimeout performs resource cleanup with a specified timeout.
//
// It creates a context with the given timeout to ensure that the cleanup
// process does not exceed the allocated time limit. The function attempts
// to release all resources used by the application and logs the outcome.
//
// Parameters:
//   - timeout: The maximum duration allowed for the cleanup process.
func cleanupResourcesWithTimeout(timeout time.Duration) {
	// Create a context with the specified timeout for the cleanup operation
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt to close and cleanup all resources
	if err := bootstrap.Close(ctx); err != nil {
		// Log an error message if the cleanup fails
		log.Printf("❌ Resource cleanup failed: %v", err)
	} else {
		// Log a success message if the cleanup completes successfully
		log.Println("✅ Resource cleanup completed successfully")
	}
}

// handlePanic is a panic handler that logs the panic error and stack trace.
//
// When a panic occurs, this function is called with the panic value as an argument.
// The function logs the panic error and stack trace using the configured logger
// service. If the logger service is not available, it falls back to the standard
// logger.
//
// Parameters:
//   - r: The panic value passed to the panic handler.
func handlePanic(r interface{}) {
	if resource.LoggerService != nil {
		resource.LoggerService.Error("Application panic recovered",
			zap.Any("panic", r),
			zap.String("stack", string(debug.Stack())),
		)
	} else {
		log.Printf("Application panic recovered: %v\n", r)
		log.Printf("Stack trace: %s\n", string(debug.Stack()))
	}
}
