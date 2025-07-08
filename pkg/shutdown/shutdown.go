package shutdown

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"
	"go.uber.org/zap"
)

var _ Hook = (*hook)(nil)

// Config holds the configuration for shutdown behavior
type Config struct {
	// TaskTimeout is the timeout for each individual cleanup task
	TaskTimeout time.Duration
	// TotalTimeout is the total timeout for all cleanup tasks
	TotalTimeout time.Duration
}

// DefaultConfig returns the default shutdown configuration
func DefaultConfig() Config {
	return Config{
		TaskTimeout:  5 * time.Second,
		TotalTimeout: 30 * time.Second,
	}
}

// CleanupFunc represents a cleanup function that can receive context
type CleanupFunc func(ctx context.Context) error

// LegacyCleanupFunc represents a legacy cleanup function without context
type LegacyCleanupFunc func()

type hook struct {
	signalChan chan os.Signal
	mu         sync.Mutex
	config     Config
}

type Hook interface {
	// WithSignals returns a Hook that will also listen for the provided signals.
	//
	// The returned Hook will listen for the signals in addition to the signals
	// already being listened to. If a signal is received, the functions passed
	// to the Close method will be executed in sequence.
	WithSignals(signals ...syscall.Signal) Hook

	// Close executes the legacy cleanup functions when a signal is received.
	// This method is kept for backward compatibility.
	//
	// It is safe to call Close multiple times from different goroutines. The
	// functions will be executed in the order they were passed to the last call
	// to Close.
	Close(funcs ...LegacyCleanupFunc)

	// CloseWithContext executes the cleanup functions with context when a signal is received.
	// This is the preferred method as it provides better error handling and timeout control.
	//
	// Returns a slice of errors from failed cleanup tasks.
	CloseWithContext(funcs ...CleanupFunc) []error
}

// NewHook creates and returns a new Hook that listens for SIGINT and SIGTERM signals.
// The returned Hook uses a channel to receive operating system signals and executes
// functions passed to the Close method in sequence when a signal is received.
func NewHook() Hook {
	h := &hook{
		signalChan: make(chan os.Signal, 1), // Channel for receiving OS signals
		config:     DefaultConfig(),         // Use default configuration
	}
	// Listen for SIGINT and SIGTERM signals
	return h.WithSignals(syscall.SIGINT, syscall.SIGTERM)
}

// WithSignals returns a Hook that will also listen for the provided signals.
//
// The returned Hook will listen for the signals in addition to the signals
// already being listened to. If a signal is received, the functions passed
// to the Close method will be executed in sequence.
func (h *hook) WithSignals(signals ...syscall.Signal) Hook {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Notify the signal channel for each signal
	for _, s := range signals {
		signal.Notify(h.signalChan, s)
	}
	return h
}

// logInfo logs an info message using the structured logger if available, otherwise falls back to standard log
func logInfo(msg string, fields ...zap.Field) {
	if resource.LoggerService != nil {
		resource.LoggerService.Info(msg, fields...)
	} else {
		log.Println("INFO:", msg)
	}
}

// logError logs an error message using the structured logger if available, otherwise falls back to standard log
func logError(msg string, fields ...zap.Field) {
	if resource.LoggerService != nil {
		resource.LoggerService.Error(msg, fields...)
	} else {
		log.Println("ERROR:", msg)
	}
}

// logWarn logs a warning message using the structured logger if available, otherwise falls back to standard log
func logWarn(msg string, fields ...zap.Field) {
	if resource.LoggerService != nil {
		resource.LoggerService.Warn(msg, fields...)
	} else {
		log.Println("WARN:", msg)
	}
}

// Close executes the legacy cleanup functions when a signal is received.
// This method is kept for backward compatibility.
//
// It is safe to call Close multiple times from different goroutines. The
// functions will be executed in the order they were passed to the last call
// to Close.
func (h *hook) Close(funcs ...LegacyCleanupFunc) {
	// Convert legacy functions to context-aware functions
	contextFuncs := make([]CleanupFunc, len(funcs))
	for i, f := range funcs {
		contextFuncs[i] = func(ctx context.Context) error {
			f()
			return nil
		}
	}

	// Use the new implementation
	h.CloseWithContext(contextFuncs...)
}

// CloseWithContext executes the cleanup functions with context when a signal is received.
// This is the preferred method as it provides better error handling and timeout control.
//
// Returns a slice of errors from failed cleanup tasks.
func (h *hook) CloseWithContext(funcs ...CleanupFunc) []error {
	// Receive the signal that triggered the shutdown
	sig := <-h.signalChan
	logInfo("🛑 Received shutdown signal", zap.String("signal", sig.String()))

	// Stop listening for signals to prevent the program from exiting immediately
	signal.Stop(h.signalChan)

	// Create a context with a timeout for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), h.config.TotalTimeout)
	defer cancel()

	// Track errors from cleanup tasks
	var errorsMu sync.Mutex
	var cleanupErrors []error

	// Use a WaitGroup to ensure all cleanup tasks complete
	var wg sync.WaitGroup
	for i, f := range funcs {
		wg.Add(1)
		go func(taskIndex int, cleanup CleanupFunc) {
			defer wg.Done()

			// Set a timeout for each cleanup task
			taskCtx, taskCancel := context.WithTimeout(shutdownCtx, h.config.TaskTimeout)
			defer taskCancel()

			// Execute the cleanup task
			taskErr := h.executeCleanupTask(taskCtx, taskIndex, cleanup)
			if taskErr != nil {
				errorsMu.Lock()
				cleanupErrors = append(cleanupErrors, taskErr)
				errorsMu.Unlock()
			}
		}(i, f)
	}

	// Wait for all cleanup tasks to complete or the total timeout to be reached
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		if len(cleanupErrors) == 0 {
			logInfo("🎉 All cleanup tasks completed successfully")
		} else {
			logWarn("⚠️ Some cleanup tasks failed", zap.Int("failed_count", len(cleanupErrors)))
		}
	case <-shutdownCtx.Done():
		logError("⏰ Shutdown timeout reached, forcing exit",
			zap.Duration("timeout", h.config.TotalTimeout))
	}

	return cleanupErrors
}

// executeCleanupTask executes a single cleanup task with proper error handling and timeout
func (h *hook) executeCleanupTask(ctx context.Context, taskIndex int, cleanup CleanupFunc) error {
	// Create a channel to receive the result of the cleanup task
	resultChan := make(chan error, 1)

	// Execute the cleanup task in a separate goroutine
	go func() {
		defer func() {
			// Recover from any panic in the cleanup function
			if r := recover(); r != nil {
				err := fmt.Errorf("cleanup task %d panicked: %v", taskIndex, r)
				logError("Cleanup task panicked",
					zap.Int("task_index", taskIndex),
					zap.Any("panic", r))
				resultChan <- err
			}
		}()

		// Execute the cleanup function
		err := cleanup(ctx)
		resultChan <- err
	}()

	// Wait for the cleanup task to complete or timeout
	select {
	case err := <-resultChan:
		if err != nil {
			logError("Cleanup task failed",
				zap.Int("task_index", taskIndex),
				zap.Error(err))
			return fmt.Errorf("cleanup task %d failed: %w", taskIndex, err)
		}
		//logInfo("🛑 Cleanup task completed successfully",
		//	zap.Int("task_index", taskIndex))
		return nil

	case <-ctx.Done():
		err := fmt.Errorf("cleanup task %d timeout after %v", taskIndex, h.config.TaskTimeout)
		logError("Cleanup task timeout",
			zap.Int("task_index", taskIndex),
			zap.Duration("timeout", h.config.TaskTimeout))
		return err
	}
}
