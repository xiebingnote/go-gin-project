package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForTDengine initializes a test logger for testing purposes
func setupTestLoggerForTDengine() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestTDengineConfig initializes a test configuration for TDengine
func setupTestTDengineConfig() {
	config.TDengineConfig = &config.TDengineConfigEntry{}
	config.TDengineConfig.TDengine.Host = "127.0.0.1"
	config.TDengineConfig.TDengine.Port = 6041
	config.TDengineConfig.TDengine.UserName = "root"
	config.TDengineConfig.TDengine.PassWord = "taosdata"
	config.TDengineConfig.TDengine.Database = "test"
	config.TDengineConfig.TDengine.ConnectTimeout = 10000 * time.Millisecond
	config.TDengineConfig.TDengine.ReadTimeout = 30000 * time.Millisecond
	config.TDengineConfig.TDengine.WriteTimeout = 30000 * time.Millisecond
	config.TDengineConfig.TDengine.MaxOpenConns = 10
	config.TDengineConfig.TDengine.MaxIdleConns = 5
	config.TDengineConfig.TDengine.ConnMaxLifetime = 3600000 * time.Millisecond
	config.TDengineConfig.TDengine.ConnMaxIdleTime = 300000 * time.Millisecond
}

// TestValidateTDengineDependencies tests the function validateTDengineDependencies
//
// It tests the following scenarios:
// - nil config
// - nil logger service
// - empty host
// - invalid port (zero or too high)
// - empty username
// - empty database
// - invalid connect timeout
// - invalid max open connections
// - max idle greater than max open
// - valid config
//
// The test uses a test logger and a test configuration. The test logger is
// initialized using the function setupTestLoggerForTDengine. The test
// configuration is initialized using the function setupTestTDengineConfig.
//
// For each test, the setupConfig function is called to set up the test
// configuration. The function validateTDengineDependencies is then called and
// the error is checked. If an error is expected, the test checks that the error
// is not nil and that the error message matches the expected error message. If
// no error is expected, the test checks that the error is nil.
func TestValidateTDengineDependencies(t *testing.T) {
	setupTestLoggerForTDengine()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.TDengineConfig = nil
			},
			expectError: true,
			errorMsg:    "tdengine configuration is not initialized",
		},
		{
			name: "nil logger service",
			setupConfig: func() {
				setupTestTDengineConfig()
				resource.LoggerService = nil
			},
			expectError: true,
			errorMsg:    "logger service is not initialized",
		},
		{
			name: "empty host",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.Host = ""
			},
			expectError: true,
			errorMsg:    "tdengine host is not configured",
		},
		{
			name: "invalid port - zero",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.Port = 0
			},
			expectError: true,
			errorMsg:    "invalid tdengine port: 0, must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.Port = 70000
			},
			expectError: true,
			errorMsg:    "invalid tdengine port: 70000, must be between 1 and 65535",
		},
		{
			name: "empty username",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.UserName = ""
			},
			expectError: true,
			errorMsg:    "tdengine username is not configured",
		},
		{
			name: "empty database",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.Database = ""
			},
			expectError: true,
			errorMsg:    "tdengine database is not configured",
		},
		{
			name: "invalid connect timeout",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.ConnectTimeout = -1 * time.Second
			},
			expectError: true,
			errorMsg:    "invalid connect timeout: -1s, must be non-negative",
		},
		{
			name: "invalid max open connections",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.MaxOpenConns = -1
			},
			expectError: true,
			errorMsg:    "invalid max open connections: -1, must be non-negative",
		},
		{
			name: "max idle greater than max open",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
				config.TDengineConfig.TDengine.MaxOpenConns = 5
				config.TDengineConfig.TDengine.MaxIdleConns = 10
			},
			expectError: true,
			errorMsg:    "max idle connections (10) cannot be greater than max open connections (5)",
		},
		{
			name: "valid config",
			setupConfig: func() {
				setupTestLoggerForTDengine()
				setupTestTDengineConfig()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			err := validateTDengineDependencies()

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

// TestBuildTDengineDSN tests the `buildTDengineDSN` function.
//
// The function takes a `config.TDengineConfigEntry` and returns a DSN string
// suitable for connecting to a TDengine database. The resulting DSN string
// should be valid for the `github.com/taosdata/driver-go` package.
//
// The test cases cover the following scenarios:
//
// 1. Basic DSN with no timeouts.
// 2. DSN with timeouts specified.
func TestBuildTDengineDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   func() *config.TDengineConfigEntry
		expected string
	}{
		{
			name: "basic dsn",
			config: func() *config.TDengineConfigEntry {
				cfg := &config.TDengineConfigEntry{}
				cfg.TDengine.Host = "localhost"
				cfg.TDengine.Port = 6041
				cfg.TDengine.UserName = "root"
				cfg.TDengine.PassWord = "taosdata"
				cfg.TDengine.Database = "test"
				return cfg
			},
			expected: "root:taosdata@ws(localhost:6041)/test",
		},
		{
			name: "dsn with timeouts",
			config: func() *config.TDengineConfigEntry {
				cfg := &config.TDengineConfigEntry{}
				cfg.TDengine.Host = "localhost"
				cfg.TDengine.Port = 6041
				cfg.TDengine.UserName = "root"
				cfg.TDengine.PassWord = "taosdata"
				cfg.TDengine.Database = "test"
				cfg.TDengine.ConnectTimeout = 10 * time.Second
				cfg.TDengine.ReadTimeout = 30 * time.Second
				return cfg
			},
			expected: "root:taosdata@ws(localhost:6041)/test?readTimeout=30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config()
			result := buildTDengineDSN(cfg)
			if result != tt.expected {
				t.Errorf("Expected DSN '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCloseTDengine_NoClient tests that CloseTDengine does not return an error
// when called with no TDengine client.
func TestCloseTDengine_NoClient(t *testing.T) {
	setupTestLoggerForTDengine()

	// Ensure no client is set
	resource.TDengineClient = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseTDengine(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no client, got: %v", err)
	}
}

// TestInitTDengine_WithValidConfig tests the `InitTDengine` function with valid
// configuration.
//
// The test will only pass if TDengine is actually running. The test is skipped
// if TDengine is not available.
//
// The test cases cover the following scenarios:
//
//  1. Valid configuration - the test will panic if the `InitTDengine` function
//     returns an error.
func TestInitTDengine_WithValidConfig(t *testing.T) {
	setupTestLoggerForTDengine()
	setupTestTDengineConfig()

	// This test will only pass if TDengine is actually running
	// Skip if TDengine is not available
	t.Skip("Skipping integration test - requires running TDengine instance and driver")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitTDengine panicked: %v", r)
		}
	}()

	InitTDengine(ctx)

	// Clean up
	if resource.TDengineClient != nil {
		_ = CloseTDengine(ctx)
	}
}
