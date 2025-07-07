package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/pkg/logger"

	"go.uber.org/zap"
)

// InitLogger initializes the LoggerService with comprehensive configuration.
//
// This function creates a production-ready logger with proper validation,
// directory setup, and error handling. The logger is configured based on
// the application configuration and supports structured logging.
//
// Parameters:
//   - ctx: Context for the initialization, used for timeouts and cancellation
func InitLogger(ctx context.Context) {
	if err := InitLoggerService(ctx); err != nil {
		// Use a temporary logger for error reporting before the main logger is ready
		tempLogger, _ := zap.NewDevelopment()
		if tempLogger != nil {
			tempLogger.Error(fmt.Sprintf("failed to initialize logger service: %v", err))
		}
		panic(fmt.Sprintf("logger service initialization failed: %v", err))
	}
}

// InitLoggerService initializes the logger service with comprehensive validation.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the logger initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates configuration and dependencies
// 2. Creates and validates log directories
// 3. Configures and creates the logger
// 4. Stores the logger in global resource
func InitLoggerService(ctx context.Context) error {
	// Validate dependencies and configuration
	if err := validateLoggerDependencies(); err != nil {
		return fmt.Errorf("logger dependencies validation failed: %w", err)
	}

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create and validate log directories
	if err := createLogDirectories(initCtx); err != nil {
		return fmt.Errorf("failed to create log directories: %w", err)
	}

	// Create and configure the logger
	loggerInstance, err := createLoggerInstance(initCtx)
	if err != nil {
		return fmt.Errorf("failed to create logger instance: %w", err)
	}

	// Store the logger in the resource package
	resource.LoggerService = loggerInstance

	// Log successful initialization
	resource.LoggerService.Info("✅ logger service initialized successfully",
		zap.String("log_dir", config.LogConfig.Log.LogDir),
		zap.String("log_level", config.LogConfig.Log.DefaultLevel),
		zap.String("version", config.ServerConfig.Version.Version),
	)

	return nil
}

// validateLoggerDependencies validates all required dependencies for logger initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateLoggerDependencies() error {
	// Check if log configuration is loaded
	if config.LogConfig == nil {
		return fmt.Errorf("log configuration is not initialized")
	}

	// Check if server configuration is loaded (needed for version info)
	if config.ServerConfig == nil {
		return fmt.Errorf("server configuration is not initialized")
	}

	cfg := &config.LogConfig.Log

	// Validate log directory
	if cfg.LogDir == "" {
		return fmt.Errorf("log directory is not configured")
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	isValidLevel := false
	for _, level := range validLevels {
		if cfg.DefaultLevel == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error", cfg.DefaultLevel)
	}

	// Validate log file names
	if cfg.LogFileDebug == "" {
		return fmt.Errorf("debug log file name is not configured")
	}
	if cfg.LogFileInfo == "" {
		return fmt.Errorf("info log file name is not configured")
	}
	if cfg.LogFileWarn == "" {
		return fmt.Errorf("warn log file name is not configured")
	}
	if cfg.LogFileError == "" {
		return fmt.Errorf("error log file name is not configured")
	}

	// Validate log rotation settings
	if cfg.MaxSize <= 0 {
		return fmt.Errorf("invalid max size: %d MB, must be greater than 0", cfg.MaxSize)
	}
	if cfg.MaxAge <= 0 {
		return fmt.Errorf("invalid max age: %d days, must be greater than 0", cfg.MaxAge)
	}
	if cfg.MaxBackups <= 0 {
		return fmt.Errorf("invalid max backups: %d, must be greater than 0", cfg.MaxBackups)
	}

	// Validate version configuration
	if config.ServerConfig.Version.Version == "" {
		return fmt.Errorf("application version is not configured")
	}

	return nil
}

// createLogDirectories creates and validates log directories.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - error: An error if directory creation fails, nil otherwise
func createLogDirectories(_ context.Context) error {
	logDir := config.LogConfig.Log.LogDir

	// Check if directory already exists
	if info, err := os.Stat(logDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("log path exists but is not a directory: %s", logDir)
		}
		// Directory exists, check permissions
		return validateDirectoryPermissions(logDir)
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Validate the created directory
	if err := validateDirectoryPermissions(logDir); err != nil {
		return fmt.Errorf("log directory validation failed: %w", err)
	}

	return nil
}

// validateDirectoryPermissions validates that the log directory has proper permissions.
//
// Parameters:
//   - dirPath: The directory path to validate
//
// Returns:
//   - error: An error if validation fails, nil otherwise
func validateDirectoryPermissions(dirPath string) error {
	// Test write permissions by creating a temporary file
	testFile := filepath.Join(dirPath, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("log directory is not writable: %s", dirPath)
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to remove test file %s: %v\n", testFile, err)
	}

	return nil
}

// createLoggerInstance creates and configures the logger instance.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - *zap.Logger: The created logger instance
//   - error: An error if logger creation fails, nil otherwise
func createLoggerInstance(ctx context.Context) (*zap.Logger, error) {
	cfg := &config.LogConfig.Log

	// Prepare logger options based on configuration
	var options []logger.Option

	// Set log level based on configuration
	switch cfg.DefaultLevel {
	case "debug":
		options = append(options, logger.WithDebugLevel())
	case "info":
		// Info is the default level, no need to set explicitly
	case "warn":
		options = append(options, logger.WithWarnLevel())
	case "error":
		options = append(options, logger.WithErrorLevel())
	}

	// Set log directory
	options = append(options, logger.WithLogDir(cfg.LogDir))

	// Add application version field
	options = append(options, logger.WithField("version", config.ServerConfig.Version.Version))

	// Add environment field if available
	if env := os.Getenv("GO_ENV"); env != "" {
		options = append(options, logger.WithField("environment", env))
	}

	// Add hostname field
	if hostname, err := os.Hostname(); err == nil {
		options = append(options, logger.WithField("hostname", hostname))
	}

	// Disable console logging in production
	if os.Getenv("GO_ENV") == "production" {
		options = append(options, logger.WithDisableConsole())
	}

	// Create logger with timeout
	done := make(chan struct{})
	var loggerInstance *zap.Logger
	var err error

	go func() {
		defer close(done)
		loggerInstance, err = logger.NewJsonLogger(options...)
	}()

	// Wait for logger creation or timeout
	select {
	case <-done:
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("logger creation timeout")
	}

	return loggerInstance, nil
}

// CloseLogger closes the logger service and flushes any pending log entries.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if the logger is initialized
// 2. Flushes any pending log entries
// 3. Clears the global resource reference
func CloseLogger(ctx context.Context) error {
	if resource.LoggerService == nil {
		// Use standard log for this message since logger is not available
		log.Println("logger service is not initialized, nothing to close")
		return nil
	}

	// Create timeout context for close operation
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Flush pending log entries
	done := make(chan error, 1)
	go func() {
		defer close(done)

		// Sync the logger to flush any pending entries
		if err := resource.LoggerService.Sync(); err != nil {
			// Check if this is a common sync error that can be safely ignored
			if !isSyncErrorIgnorable(err) {
				log.Printf("Warning: logger sync failed during close: %v", err)
			}
		}

		done <- nil
	}()

	// Wait for sync operation or timeout
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to close logger service: %w", err)
		}
	case <-closeCtx.Done():
		fmt.Println("Warning: logger close timeout, proceeding anyway")
	}

	// Clear the global reference
	resource.LoggerService = nil

	return nil
}

// FlushLogger flushes any pending log entries without closing the logger.
//
// Returns:
//   - error: An error if the flush operation fails, nil otherwise
func FlushLogger() error {
	if resource.LoggerService == nil {
		return fmt.Errorf("logger service is not initialized")
	}

	if err := resource.LoggerService.Sync(); err != nil {
		// Check if this is a common sync error that can be safely ignored
		if isSyncErrorIgnorable(err) {
			log.Printf("Info: logger flush completed with expected warnings: %v", err)
			return nil
		}
		return fmt.Errorf("failed to flush logger: %w", err)
	}

	return nil
}

// GetLoggerLevel returns the current log level as a string.
//
// Returns:
//   - string: The current log level
func GetLoggerLevel() string {
	if config.LogConfig == nil {
		return "unknown"
	}
	return config.LogConfig.Log.DefaultLevel
}

// IsLoggerInitialized checks if the logger service is initialized.
//
// Returns:
//   - bool: True if logger is initialized, false otherwise
func IsLoggerInitialized() bool {
	return resource.LoggerService != nil
}

// isSyncErrorIgnorable checks if a sync error can be safely ignored.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error can be ignored, false otherwise
//
// Common ignorable sync errors include:
// - "sync /dev/stdout: bad file descriptor" - stdout already closed
// - "sync /dev/stderr: bad file descriptor" - stderr already closed
// - "sync /dev/stdout: inappropriate ioctl for device" - stdout is not a regular file
// - "sync /dev/stderr: inappropriate ioctl for device" - stderr is not a regular file
func isSyncErrorIgnorable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for common ignorable sync errors
	ignorablePatterns := []string{
		"sync /dev/stdout: bad file descriptor",
		"sync /dev/stderr: bad file descriptor",
		"sync /dev/stdout: inappropriate ioctl for device",
		"sync /dev/stderr: inappropriate ioctl for device",
		"sync /dev/stdout: operation not supported",
		"sync /dev/stderr: operation not supported",
	}

	for _, pattern := range ignorablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
