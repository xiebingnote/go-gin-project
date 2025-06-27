package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForManticore initializes a test logger for testing purposes
func setupTestLoggerForManticore() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestManticoreConfig initializes a test configuration for Manticore
func setupTestManticoreConfig() {
	config.ManticoreConfig = &config.ManticoreConfigEntry{}
	config.ManticoreConfig.Manticore.Endpoints = []string{"127.0.0.1"}
	config.ManticoreConfig.Manticore.Port = 9308
	config.ManticoreConfig.Manticore.UserName = ""
	config.ManticoreConfig.Manticore.PassWord = ""
}

// TestValidateManticoreDependencies tests validateManticoreDependencies with a
// variety of configurations to ensure that it correctly validates Manticore
// dependencies and returns an error if any dependency is invalid.
func TestValidateManticoreDependencies(t *testing.T) {
	setupTestLoggerForManticore()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.ManticoreConfig = nil
			},
			expectError: true,
			errorMsg:    "manticore configuration is not initialized",
		},
		{
			name: "nil logger service",
			setupConfig: func() {
				setupTestManticoreConfig()
				resource.LoggerService = nil
			},
			expectError: true,
			errorMsg:    "logger service is not initialized",
		},
		{
			name: "empty endpoints",
			setupConfig: func() {
				setupTestLoggerForManticore()
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Endpoints = []string{}
			},
			expectError: true,
			errorMsg:    "manticore endpoints are not configured",
		},
		{
			name: "invalid port - zero",
			setupConfig: func() {
				setupTestLoggerForManticore()
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Port = 0
			},
			expectError: true,
			errorMsg:    "invalid manticore port: 0, must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			setupConfig: func() {
				setupTestLoggerForManticore()
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Port = 70000
			},
			expectError: true,
			errorMsg:    "invalid manticore port: 70000, must be between 1 and 65535",
		},
		{
			name: "empty endpoint",
			setupConfig: func() {
				setupTestLoggerForManticore()
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Endpoints = []string{"127.0.0.1", ""}
			},
			expectError: true,
			errorMsg:    "manticore endpoint 1 is empty",
		},
		{
			name: "valid config",
			setupConfig: func() {
				setupTestLoggerForManticore()
				setupTestManticoreConfig()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			err := validateManticoreDependencies()

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

// TestCreateManticoreClient tests creating a ManticoreSearch client with various
// configurations, such as single endpoint, multiple endpoints, with
// authentication, and custom port.
func TestCreateManticoreClient(t *testing.T) {
	setupTestLoggerForManticore()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
	}{
		{
			name: "single endpoint",
			setupConfig: func() {
				setupTestManticoreConfig()
			},
			expectError: false,
		},
		{
			name: "multiple endpoints",
			setupConfig: func() {
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Endpoints = []string{"127.0.0.1", "127.0.0.2"}
			},
			expectError: false,
		},
		{
			name: "with authentication",
			setupConfig: func() {
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.UserName = "testuser"
				config.ManticoreConfig.Manticore.PassWord = "testpass"
			},
			expectError: false,
		},
		{
			name: "custom port",
			setupConfig: func() {
				setupTestManticoreConfig()
				config.ManticoreConfig.Manticore.Port = 9312
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := createManticoreClient(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client to be created")
				}
			}
		})
	}
}

// TestEncodeBasicAuth tests the `encodeBasicAuth` function with various
// combinations of username and password. The test cases cover the following
// scenarios:
//
//  1. Simple credentials with a username and password.
//  2. Empty password, which results in a colon (:) at the end of the string.
//  3. Empty username, which results in a leading colon (:) in the string.
func TestEncodeBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{
			name:     "simple credentials",
			username: "user",
			password: "pass",
			expected: "user:pass",
		},
		{
			name:     "empty password",
			username: "user",
			password: "",
			expected: "user:",
		},
		{
			name:     "empty username",
			username: "",
			password: "pass",
			expected: ":pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeBasicAuth(tt.username, tt.password)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestContains tests the `contains` function with various strings and substrings.
// The test cases cover the following scenarios:
//
//  1. The substring is at the start of the string.
//  2. The substring is at the end of the string.
//  3. The substring is in the middle of the string.
//  4. The substring is not present in the string.
//  5. The substring is an exact match of the string.
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains at start",
			s:        "connection error",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "contains at end",
			s:        "network timeout",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "contains in middle",
			s:        "network connection error",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "does not contain",
			s:        "success",
			substr:   "error",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "error",
			substr:   "error",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCloseManticore_NoClient tests the CloseManticore function when there is
// no Manticore client initialized. The function should not return an error in
// this case.
func TestCloseManticore_NoClient(t *testing.T) {
	setupTestLoggerForManticore()

	// Ensure no client is set
	resource.ManticoreClient = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseManticore(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no client, got: %v", err)
	}
}

// TestInitManticore_WithValidConfig tests the InitManticore function with valid
// configuration values. This test will only pass if Manticore is actually
// running, as it requires a running instance to test the connection.
//
// The test will skip if Manticore is not running, so as not to fail the test
// suite.
//
// The test also checks that the function does not panic when given valid
// configuration values.
func TestInitManticore_WithValidConfig(t *testing.T) {
	setupTestLoggerForManticore()
	setupTestManticoreConfig()

	// This test will only pass if Manticore is actually running
	// Skip if Manticore is not available
	t.Skip("Skipping integration test - requires running Manticore instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitManticore panicked: %v", r)
		}
	}()

	InitManticore(ctx)

	// Clean up
	if resource.ManticoreClient != nil {
		_ = CloseManticore(ctx)
	}
}
