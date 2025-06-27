package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerConfig initializes a test configuration for Logger
func setupTestLoggerConfig() {
	config.LogConfig = &config.LogConfigEntry{
		Log: struct {
			DefaultLevel string `toml:"DefaultLevel"`
			LogDir       string `toml:"LogDir"`
			LogFileDebug string `toml:"LogFileDebug"`
			LogFileInfo  string `toml:"LogFileInfo"`
			LogFileWarn  string `toml:"LogFileWarn"`
			LogFileError string `toml:"LogFileError"`
			MaxSize      int    `toml:"MaxSize"`
			MaxAge       int    `toml:"MaxAge"`
			MaxBackups   int    `toml:"MaxBackups"`
			LocalTime    bool   `toml:"LocalTime"`
			Compress     bool   `toml:"Compress"`
		}{
			DefaultLevel: "info",
			LogDir:       "./test_logs",
			LogFileDebug: "debug.log",
			LogFileInfo:  "info.log",
			LogFileWarn:  "warn.log",
			LogFileError: "error.log",
			MaxSize:      100,
			MaxAge:       30,
			MaxBackups:   10,
			LocalTime:    true,
			Compress:     false,
		},
	}

	config.ServerConfig = &config.ServerConfigEntry{
		Version: struct {
			Version string `toml:"Version"`
		}{
			Version: "1.0.0-test",
		},
	}
}

// cleanupTestLogDir removes the test log directory
func cleanupTestLogDir() {
	if config.LogConfig != nil {
		os.RemoveAll(config.LogConfig.Log.LogDir)
	}
}

// TestValidateLoggerDependencies tests the validateLoggerDependencies function.
//
// The function is tested with various invalid and valid configurations.
//
// The test cases are as follows:
//
//   - nil log config: The log configuration is set to nil.
//   - nil server config: The server configuration is set to nil.
//   - empty log directory: The log directory is set to an empty string.
//   - invalid log level: The log level is set to an invalid value.
//   - empty debug log file: The debug log file is set to an empty string.
//   - invalid max size: The max size is set to 0.
//   - invalid max age: The max age is set to 0.
//   - invalid max backups: The max backups is set to 0.
//   - empty version: The application version is set to an empty string.
//   - valid config: The configuration is set to a valid configuration.
func TestValidateLoggerDependencies(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil log config",
			setupConfig: func() {
				config.LogConfig = nil
				config.ServerConfig = &config.ServerConfigEntry{
					Version: struct {
						Version string `toml:"Version"`
					}{Version: "1.0.0"},
				}
			},
			expectError: true,
			errorMsg:    "log configuration is not initialized",
		},
		{
			name: "nil server config",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.ServerConfig = nil
			},
			expectError: true,
			errorMsg:    "server configuration is not initialized",
		},
		{
			name: "empty log directory",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.LogDir = ""
			},
			expectError: true,
			errorMsg:    "log directory is not configured",
		},
		{
			name: "invalid log level",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.DefaultLevel = "invalid"
			},
			expectError: true,
			errorMsg:    "invalid log level: invalid, must be one of: debug, info, warn, error",
		},
		{
			name: "empty debug log file",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.LogFileDebug = ""
			},
			expectError: true,
			errorMsg:    "debug log file name is not configured",
		},
		{
			name: "invalid max size",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.MaxSize = 0
			},
			expectError: true,
			errorMsg:    "invalid max size: 0 MB, must be greater than 0",
		},
		{
			name: "invalid max age",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.MaxAge = 0
			},
			expectError: true,
			errorMsg:    "invalid max age: 0 days, must be greater than 0",
		},
		{
			name: "invalid max backups",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.LogConfig.Log.MaxBackups = 0
			},
			expectError: true,
			errorMsg:    "invalid max backups: 0, must be greater than 0",
		},
		{
			name: "empty version",
			setupConfig: func() {
				setupTestLoggerConfig()
				config.ServerConfig.Version.Version = ""
			},
			expectError: true,
			errorMsg:    "application version is not configured",
		},
		{
			name:        "valid config",
			setupConfig: setupTestLoggerConfig,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()
			defer cleanupTestLogDir()

			err := validateLoggerDependencies()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestCreateLogDirectories tests the createLogDirectories function, which creates the log directories
// and returns an error if the creation fails. It also tests the case where the directory already
// exists.
func TestCreateLogDirectories(t *testing.T) {
	setupTestLoggerConfig()
	defer cleanupTestLogDir()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test directory creation
	err := createLogDirectories(ctx)
	if err != nil {
		t.Errorf("Expected no error creating log directories, got: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(config.LogConfig.Log.LogDir); os.IsNotExist(err) {
		t.Errorf("Log directory was not created: %s", config.LogConfig.Log.LogDir)
	}

	// Test with existing directory
	err = createLogDirectories(ctx)
	if err != nil {
		t.Errorf("Expected no error with existing directory, got: %v", err)
	}
}

// TestValidateDirectoryPermissions tests the validateDirectoryPermissions function, which checks if a directory has write
// permissions. It tests both the case where the directory has valid permissions and the case where the directory is
// read-only (if not running as root).
func TestValidateDirectoryPermissions(t *testing.T) {
	setupTestLoggerConfig()
	defer cleanupTestLogDir()

	// Create test directory
	testDir := config.LogConfig.Log.LogDir
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Test valid permissions
	err = validateDirectoryPermissions(testDir)
	if err != nil {
		t.Errorf("Expected no error with valid permissions, got: %v", err)
	}

	// Test with read-only directory (if not running as root)
	if os.Getuid() != 0 {
		readOnlyDir := filepath.Join(testDir, "readonly")
		err = os.MkdirAll(readOnlyDir, 0444)
		if err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		err = validateDirectoryPermissions(readOnlyDir)
		if err == nil {
			t.Errorf("Expected error with read-only directory, got none")
		}
	}
}

// TestGetLoggerLevel tests the GetLoggerLevel function, which returns the
// configured log level. It tests both the case where the config is valid and
// the case where the config is nil.
func TestGetLoggerLevel(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected string
	}{
		{
			name: "with valid config",
			setup: func() {
				setupTestLoggerConfig()
			},
			expected: "info",
		},
		{
			name: "with nil config",
			setup: func() {
				config.LogConfig = nil
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer cleanupTestLogDir()

			result := GetLoggerLevel()
			if result != tt.expected {
				t.Errorf("Expected log level '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestIsLoggerInitialized tests the IsLoggerInitialized function, which returns
// true if the logger service has been initialized and false otherwise.
//
// The test case creates a logger and sets it on the resource, and then verifies
// that IsLoggerInitialized returns true. It then sets the logger to nil and
// verifies that IsLoggerInitialized returns false.
func TestIsLoggerInitialized(t *testing.T) {
	// Test with no logger
	resource.LoggerService = nil
	if IsLoggerInitialized() {
		t.Errorf("Expected false when logger is not initialized")
	}

	// Test with logger
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
	if !IsLoggerInitialized() {
		t.Errorf("Expected true when logger is initialized")
	}

	// Clean up
	resource.LoggerService = nil
}

// TestFlushLogger verifies the behavior of the FlushLogger function.
//
// The test case covers the following scenarios:
// 1. When the logger is uninitialized, it expects an error when calling FlushLogger.
// 2. When the logger is initialized, it expects no error when calling FlushLogger.
//
// It sets the logger service to nil to simulate the uninitialized state, and
// uses a development logger to simulate the initialized state.
func TestFlushLogger(t *testing.T) {
	// Test with no logger
	resource.LoggerService = nil
	err := FlushLogger()
	if err == nil {
		t.Errorf("Expected error when flushing uninitialized logger")
	}

	// Test with logger
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
	err = FlushLogger()
	if err != nil {
		t.Errorf("Expected no error when flushing initialized logger, got: %v", err)
	}

	// Clean up
	resource.LoggerService = nil
}

// TestCloseLogger_NoLogger tests the CloseLogger function when no logger is set.
//
// The test sets the logger service to nil to simulate the uninitialized state,
// and verifies that no error is returned when calling CloseLogger.
func TestCloseLogger_NoLogger(t *testing.T) {
	// Ensure no logger is set
	resource.LoggerService = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseLogger(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no logger, got: %v", err)
	}
}

// TestInitLogger_WithValidConfig tests the InitLogger function with a valid
// configuration by verifying that it initializes the logger properly.
//
// The test sets up a valid test configuration and calls InitLogger. It then
// verifies that the logger is initialized and that no error is returned.
//
// Finally, the test cleans up the logger to prevent memory leaks.
func TestInitLogger_WithValidConfig(t *testing.T) {
	setupTestLoggerConfig()
	defer cleanupTestLogDir()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitLogger panicked: %v", r)
		}
	}()

	InitLogger(ctx)

	// Verify logger was created
	if !IsLoggerInitialized() {
		t.Errorf("Expected logger to be initialized")
	}

	// Clean up
	if resource.LoggerService != nil {
		_ = CloseLogger(ctx)
	}
}
