package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForEtcd initializes a test logger for testing purposes
func setupTestLoggerForEtcd() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestEtcdConfig initializes a test configuration for Etcd
func setupTestEtcdConfig() *config.EtcdConfigEntry {
	return &config.EtcdConfigEntry{
		Etcd: struct {
			Endpoints   []string      `toml:"Endpoints"`
			DialTimeout time.Duration `toml:"DialTimeout"`
			Username    string        `toml:"UserName"`
			Password    string        `toml:"PassWord"`
		}{
			Endpoints:   []string{"http://localhost:2379"},
			DialTimeout: 5 * time.Second,
			Username:    "root",
			Password:    "password",
		},
	}
}

// TestValidateEtcdConfig tests the ValidateEtcdConfig function.
//
// This test function verifies that the ValidateEtcdConfig function correctly
// identifies invalid and valid Etcd configuration scenarios. The test cases
// cover various scenarios, including nil configuration, empty endpoints,
// empty elements in the endpoints list, invalid endpoint formats, empty
// username, empty password, invalid dial timeout, and a valid configuration.
//
// Each test case specifies the expected outcome (error or no error) and
// ensures the error messages match the expected error messages when applicable.
func TestValidateEtcdConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.EtcdConfigEntry
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "etcd configuration is nil",
		},
		{
			name: "empty endpoints",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.Endpoints = []string{}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "etcd endpoints are empty",
		},
		{
			name: "empty endpoint in list",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.Endpoints = []string{"http://localhost:2379", ""}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "etcd endpoint[1] is empty",
		},
		{
			name: "invalid endpoint format",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.Endpoints = []string{"invalid-endpoint"}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid etcd endpoint[0]: invalid-endpoint",
		},
		{
			name: "empty username",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.Username = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "etcd username is empty",
		},
		{
			name: "empty password",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.Password = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "etcd password is empty",
		},
		{
			name: "invalid dial timeout",
			config: func() *config.EtcdConfigEntry {
				cfg := setupTestEtcdConfig()
				cfg.Etcd.DialTimeout = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid dial timeout: 0s, must be greater than 0",
		},
		{
			name:        "valid config",
			config:      setupTestEtcdConfig(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEtcdConfig(tt.config)

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

// TestIsValidEtcdEndpoint tests the function isValidEtcdEndpoint with a variety of
// endpoints to ensure that it correctly validates Etcd endpoints and returns a boolean
// indicating whether the endpoint is valid or not.
//
// The function checks the following:
//
//  1. The endpoint is not empty
//  2. The endpoint starts with http or https
//  3. The endpoint contains a host (either IP or domain)
//  4. The endpoint contains a port
//
// The test suite covers various scenarios, including valid and invalid endpoints, to
// ensure that the function behaves correctly in all cases.
func TestIsValidEtcdEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "valid http endpoint",
			endpoint: "http://localhost:2379",
			expected: true,
		},
		{
			name:     "valid https endpoint",
			endpoint: "https://localhost:2379",
			expected: true,
		},
		{
			name:     "valid endpoint with IP",
			endpoint: "http://127.0.0.1:2379",
			expected: true,
		},
		{
			name:     "valid endpoint with domain",
			endpoint: "https://etcd.example.com:2379",
			expected: true,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			expected: false,
		},
		{
			name:     "endpoint without scheme",
			endpoint: "localhost:2379",
			expected: false,
		},
		{
			name:     "endpoint without host",
			endpoint: "http://",
			expected: false,
		},
		{
			name:     "invalid scheme",
			endpoint: "ftp://localhost:2379",
			expected: false,
		},
		{
			name:     "malformed URL",
			endpoint: "http://[::1:2379",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEtcdEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for endpoint: %s", tt.expected, result, tt.endpoint)
			}
		})
	}
}

// TestConfigureEtcdClient tests the ConfigureEtcdClient function.
//
// The function configures the Etcd client options based on the provided
// configuration. The test verifies the basic configuration of the client
// and additional settings.
func TestConfigureEtcdClient(t *testing.T) {
	cfg := setupTestEtcdConfig()

	clientConfig := ConfigureEtcdClient(cfg)

	// Verify basic configuration
	if len(clientConfig.Endpoints) != len(cfg.Etcd.Endpoints) {
		t.Errorf("Expected %d endpoints, got %d", len(cfg.Etcd.Endpoints), len(clientConfig.Endpoints))
	}

	for i, endpoint := range cfg.Etcd.Endpoints {
		if clientConfig.Endpoints[i] != endpoint {
			t.Errorf("Expected endpoint[%d] to be %s, got %s", i, endpoint, clientConfig.Endpoints[i])
		}
	}

	if clientConfig.Username != cfg.Etcd.Username {
		t.Errorf("Expected username %s, got %s", cfg.Etcd.Username, clientConfig.Username)
	}

	if clientConfig.Password != cfg.Etcd.Password {
		t.Errorf("Expected password %s, got %s", cfg.Etcd.Password, clientConfig.Password)
	}

	expectedDialTimeout := cfg.Etcd.DialTimeout * time.Second
	if clientConfig.DialTimeout != expectedDialTimeout {
		t.Errorf("Expected dial timeout %v, got %v", expectedDialTimeout, clientConfig.DialTimeout)
	}

	// Verify additional settings
	if clientConfig.AutoSyncInterval != 30*time.Second {
		t.Errorf("Expected AutoSyncInterval to be 30s, got %v", clientConfig.AutoSyncInterval)
	}

	if !clientConfig.RejectOldCluster {
		t.Errorf("Expected RejectOldCluster to be true")
	}

	if !clientConfig.PermitWithoutStream {
		t.Errorf("Expected PermitWithoutStream to be true")
	}
}

// TestTestEtcdConnection_WithMockClient tests the TestEtcdConnection function
// with a mock etcd client. The test is skipped for now as it requires an
// actual etcd connection. The test should be completed once a mock etcd
// client is available.
func TestTestEtcdConnection_WithMockClient(t *testing.T) {
	// This test would require a mock etcd client
	// Skip for now as it requires actual etcd connection
	t.Skip("Skipping test - requires actual etcd connection")
}

// TestCloseEtcd_NoClient tests the CloseEtcd function when there is no
// existing etcd client connection. The test verifies that calling CloseEtcd
// with no client does not result in an error.
func TestCloseEtcd_NoClient(t *testing.T) {
	setupTestLoggerForEtcd()

	// Ensure no client is set
	resource.EtcdClient = nil

	err := CloseEtcd()
	if err != nil {
		t.Errorf("Expected no error when closing with no client, got: %v", err)
	}
}

// TestCloseEtcd_WithMockClient tests the CloseEtcd function with a mock etcd
// client. The test verifies that calling CloseEtcd with a mock client does
// not result in an error. The test is skipped for now as it requires proper
// mock setup.
func TestCloseEtcd_WithMockClient(t *testing.T) {
	setupTestLoggerForEtcd()

	// This test would require a mock etcd client
	// Skip for now as it requires proper mock setup
	t.Skip("Skipping test - requires mock etcd client setup")
}

// TestInitEtcd_WithValidConfig tests the InitEtcd function with a valid
// configuration. The test verifies that the function does not panic when
// called with a valid configuration. The test is skipped if etcd is not
// available.
//
// Note: This test requires a running etcd instance to pass.
func TestInitEtcd_WithValidConfig(t *testing.T) {
	setupTestLoggerForEtcd()

	// Set up test config
	config.EtcdConfig = setupTestEtcdConfig()

	// This test will only pass if etcd is actually running
	// Skip if etcd is not available
	t.Skip("Skipping integration test - requires running etcd instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitEtcd panicked: %v", r)
		}
	}()

	InitEtcd(ctx)

	// Clean up
	if resource.EtcdClient != nil {
		_ = CloseEtcd()
	}
}

// TestInitEtcdClient_ConfigValidation tests the InitEtcdClient function
// with an invalid configuration. The test verifies that the function returns
// an error when the configuration is invalid and that the error message
// contains the expected substring "invalid etcd configuration".
func TestInitEtcdClient_ConfigValidation(t *testing.T) {
	setupTestLoggerForEtcd()

	// Test with invalid config
	config.EtcdConfig = &config.EtcdConfigEntry{}

	err := InitEtcdClient()
	if err == nil {
		t.Errorf("Expected error with invalid config, got none")
	}

	// Verify error message contains validation failure
	expectedSubstring := "invalid etcd configuration"
	if err != nil && !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}
