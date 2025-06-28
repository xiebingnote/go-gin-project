package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// InitEtcd initializes the Etcd client with the configuration
// specified in the ./conf/etcd.toml file.
func InitEtcd(_ context.Context) {
	if err := InitEtcdClient(); err != nil {
		// Log an error message if the Etcd connection cannot be established
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize etcd: %v", err))
		panic(err.Error())
	}
}

// InitEtcdClient initializes a new Etcd client with connection pool.
//
// The function performs the following steps:
// 1. Validates the Etcd configuration
// 2. Configures the client options
// 3. Creates the Etcd client
// 4. Tests the connection
// 5. Stores the initialized client in the global resource
//
// Returns an error if the configuration is invalid, the connection cannot be established,
// or any of the steps fail.
func InitEtcdClient() error {
	cfg := config.EtcdConfig

	// Validate the Etcd configuration
	if err := ValidateEtcdConfig(cfg); err != nil {
		return fmt.Errorf("invalid etcd configuration: %w", err)
	}

	// Prepare the etcd client configuration
	clientConfig := ConfigureEtcdClient(cfg)

	// Create the etcd client
	etcdClient, err := clientv3.New(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	// Test the connection
	if err := TestEtcdConnection(etcdClient, cfg); err != nil {
		// Close the client if connection test fails to prevent resource leak
		if closeErr := etcdClient.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close etcd client during cleanup: %v", closeErr))
		}
		return fmt.Errorf("failed to test etcd connection: %w", err)
	}

	// Store the initialized client in the global resource
	resource.EtcdClient = etcdClient
	resource.LoggerService.Info("successfully connected to etcd")

	return nil
}

// ValidateEtcdConfig validates the Etcd configuration.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Endpoints
//  3. Connection settings are valid:
//     - DialTimeout
//  4. Authentication settings (optional):
//     - If username is provided, password must also be provided
//     - If password is provided, username must also be provided
func ValidateEtcdConfig(cfg *config.EtcdConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("etcd configuration is nil")
	}

	// Check required connection parameters
	if len(cfg.Etcd.Endpoints) == 0 {
		return fmt.Errorf("etcd endpoints are empty")
	}

	// Validate each endpoint
	for i, endpoint := range cfg.Etcd.Endpoints {
		if endpoint == "" {
			return fmt.Errorf("etcd endpoint[%d] is empty", i)
		}
		// Basic URL format validation
		if !isValidEtcdEndpoint(endpoint) {
			return fmt.Errorf("invalid etcd endpoint[%d]: %s", i, endpoint)
		}
	}

	// Validate authentication settings (both must be provided or both must be empty)
	hasUsername := cfg.Etcd.Username != ""
	hasPassword := cfg.Etcd.Password != ""

	if hasUsername && !hasPassword {
		return fmt.Errorf("etcd username provided but password is empty")
	}
	if hasPassword && !hasUsername {
		return fmt.Errorf("etcd password provided but username is empty")
	}

	// Check connection settings with detailed error messages
	if cfg.Etcd.DialTimeout <= 0 {
		return fmt.Errorf("invalid dial timeout: %v, must be greater than 0", cfg.Etcd.DialTimeout)
	}

	return nil
}

// ConfigureEtcdClient configures the Etcd client options.
//
// Parameters:
//   - cfg: A pointer to the Etcd configuration containing the connection settings.
//
// Returns:
//   - A configured clientv3.Config instance.
func ConfigureEtcdClient(cfg *config.EtcdConfigEntry) clientv3.Config {
	configEtcd := clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		DialTimeout: cfg.Etcd.DialTimeout * time.Second,
		// Additional client configuration
		AutoSyncInterval:     0, // Disable auto sync to prevent "no such host" errors in single-node setup
		RejectOldCluster:     true,
		PermitWithoutStream:  true,
		DialKeepAliveTime:    30 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
		MaxCallSendMsgSize:   10 * 1024 * 1024, // 10MB
		MaxCallRecvMsgSize:   10 * 1024 * 1024, // 10MB
	}

	// Only set authentication if username and password are provided
	if cfg.Etcd.Username != "" && cfg.Etcd.Password != "" {
		configEtcd.Username = cfg.Etcd.Username
		configEtcd.Password = cfg.Etcd.Password
	}

	return configEtcd
}

// TestEtcdConnection tests the Etcd connection.
//
// The function takes an Etcd client instance and configuration as parameters
// and tests the connection by executing a status check. If the check fails,
// the function returns an error.
//
// Parameters:
//   - client: An Etcd client instance to test.
//   - cfg: The Etcd configuration containing the endpoints.
//
// Returns:
//   - An error if the status check fails
//   - nil if the status check succeeds
func TestEtcdConnection(client *clientv3.Client, cfg *config.EtcdConfigEntry) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection using the first endpoint
	endpoint := cfg.Etcd.Endpoints[0]
	_, err := client.Status(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to get etcd status from %s: %w", endpoint, err)
	}

	return nil
}

// CloseEtcd closes the Etcd connection.
//
// This function attempts to close the global Etcd client connection.
// If the connection is successfully closed, it returns nil. If the client is nil,
// it also returns nil.
//
// Returns:
//   - An error if there is an issue closing the connection
//   - nil if the Etcd client is nil or the connection is closed successfully
func CloseEtcd() error {
	// Check if the global Etcd client is initialized
	if resource.EtcdClient == nil {
		// The Etcd client is nil, no connection to close
		return nil
	}

	// Attempt to close the Etcd connection
	if err := resource.EtcdClient.Close(); err != nil {
		if resource.LoggerService != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close etcd connection: %v", err))
		}
		return err
	}

	// Reset the global Etcd client to nil
	resource.EtcdClient = nil
	resource.LoggerService.Info("successfully closed etcd connection")

	return nil
}

// isValidEtcdEndpoint validates if the given string is a valid etcd endpoint format.
//
// Parameters:
//   - endpoint: The endpoint string to validate.
//
// Returns:
//   - true if the endpoint is valid, false otherwise.
func isValidEtcdEndpoint(endpoint string) bool {
	if endpoint == "" {
		return false
	}

	// Parse the URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	// Check if scheme and host are present
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}

	// Check if scheme is http or https
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}
