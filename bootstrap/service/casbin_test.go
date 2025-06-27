package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// setupTestLoggerForCasbin initializes a test logger for testing purposes
func setupTestLoggerForCasbin() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestMySQLClient creates a mock MySQL client for testing
func setupTestMySQLClient() error {
	// Create a mock GORM DB instance
	// In a real test environment, you would use a test database
	resource.MySQLClient = &gorm.DB{}
	return nil
}

// createTestCasbinConfig creates a temporary Casbin configuration file for testing
func createTestCasbinConfig() (string, error) {
	configContent := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

	tmpFile, err := os.CreateTemp("", "casbin_test_*.conf")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// TestValidateCasbinDependencies tests the ValidateCasbinDependencies function with
// various scenarios of invalid and valid dependencies.
//
// The tests cover the following scenarios:
// - Nil MySQL client
// - Nil logger service
// - Missing config file
// - Valid dependencies with a temporary config file
//
// The function is expected to return an error if any of the dependencies are not
// initialized or if the config file is not found. Otherwise, it should return nil.
func TestValidateCasbinDependencies(t *testing.T) {
	setupTestLoggerForCasbin()

	tests := []struct {
		name        string
		setupFunc   func() error
		cleanupFunc func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil mysql client",
			setupFunc: func() error {
				resource.MySQLClient = nil
				return nil
			},
			expectError: true,
			errorMsg:    "mysql client is not initialized",
		},
		{
			name: "nil logger service",
			setupFunc: func() error {
				if err := setupTestMySQLClient(); err != nil {
					return err
				}
				resource.LoggerService = nil
				return nil
			},
			cleanupFunc: func() {
				resource.MySQLClient = nil
			},
			expectError: true,
			errorMsg:    "logger service is not initialized",
		},
		{
			name: "missing config file",
			setupFunc: func() error {
				setupTestLoggerForCasbin()
				if err := setupTestMySQLClient(); err != nil {
					return err
				}
				// Set non-existent config path
				os.Setenv("CASBIN_CONFIG_PATH", "/non/existent/path.conf")
				return nil
			},
			cleanupFunc: func() {
				resource.MySQLClient = nil
				os.Unsetenv("CASBIN_CONFIG_PATH")
			},
			expectError: true,
			errorMsg:    "casbin configuration file not found: /non/existent/path.conf",
		},
		{
			name: "valid dependencies",
			setupFunc: func() error {
				setupTestLoggerForCasbin()
				if err := setupTestMySQLClient(); err != nil {
					return err
				}

				// Create temporary config file
				configPath, err := createTestCasbinConfig()
				if err != nil {
					return err
				}
				os.Setenv("CASBIN_CONFIG_PATH", configPath)
				return nil
			},
			cleanupFunc: func() {
				if configPath := os.Getenv("CASBIN_CONFIG_PATH"); configPath != "" {
					os.Remove(configPath)
					os.Unsetenv("CASBIN_CONFIG_PATH")
				}
				resource.MySQLClient = nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			err := validateCasbinDependencies()

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

// TestGetCasbinConfigPath tests that the `getCasbinConfigPath` function returns the correct casbin config path.
//
// The test cases cover the following scenarios:
//
// 1. No environment variable set: the function should return the default config path.
// 2. Environment variable set to a custom path: the function should return the custom path.
func TestGetCasbinConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default path",
			envValue: "",
			expected: "./conf/service/casbin.conf",
		},
		{
			name:     "custom path from env",
			envValue: "/custom/path/casbin.conf",
			expected: "/custom/path/casbin.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv("CASBIN_CONFIG_PATH", tt.envValue)
			} else {
				os.Unsetenv("CASBIN_CONFIG_PATH")
			}

			// Cleanup
			defer func() {
				if tt.envValue != "" {
					os.Unsetenv("CASBIN_CONFIG_PATH")
				}
			}()

			result := getCasbinConfigPath()
			if result != tt.expected {
				t.Errorf("Expected path '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCloseCasbin_NoEnforcer tests the CloseCasbin function when no enforcer is set.
//
// The test ensures that no error is returned when the enforcer is not set.
func TestCloseCasbin_NoEnforcer(t *testing.T) {
	setupTestLoggerForCasbin()

	// Ensure no enforcer is set
	resource.Enforcer = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseCasbin(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no enforcer, got: %v", err)
	}
}

// TestInitCasbinEnforcer_WithValidSetup tests the InitEnforcer function with all dependencies
// properly set up.
//
// The test ensures that no panic is triggered when the function is called with a valid
// setup.
//
// This test requires a proper test environment setup, so it is skipped if the test
// can't create a valid test environment.
func TestInitCasbinEnforcer_WithValidSetup(t *testing.T) {
	setupTestLoggerForCasbin()

	// This test will only pass if all dependencies are properly set up
	// Skip if we can't create a proper test environment
	t.Skip("Skipping integration test - requires proper test environment setup")

	if err := setupTestMySQLClient(); err != nil {
		t.Fatalf("Failed to setup test MySQL client: %v", err)
	}
	defer func() {
		resource.MySQLClient = nil
	}()

	// Create temporary config file
	configPath, err := createTestCasbinConfig()
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(configPath)

	os.Setenv("CASBIN_CONFIG_PATH", configPath)
	defer os.Unsetenv("CASBIN_CONFIG_PATH")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid setup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitEnforcer panicked: %v", r)
		}
	}()

	InitEnforcer(ctx)

	// Clean up
	if resource.Enforcer != nil {
		_ = CloseCasbin(ctx)
	}
}
