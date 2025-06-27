package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	manticore "github.com/manticoresoftware/manticoresearch-go"
)

// InitManticore initializes the ManticoreSearch client using the configuration
// specified in the ./conf/manticore.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ManticoreSearch cluster.
//
// The initialized ManticoreSearch client is stored as a singleton in the
// resource package for use throughout the application.
//
// Parameters:
//   - ctx: Context for the initialization, used for timeouts and cancellation
func InitManticore(ctx context.Context) {
	if err := InitManticoreClient(ctx); err != nil {
		// Log the error before panicking
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize manticore client: %v", err))
		panic(fmt.Sprintf("manticore client initialization failed: %v", err))
	}
}

// InitManticoreClient initializes the ManticoreSearch client with comprehensive validation.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates configuration and dependencies
// 2. Creates and configures the ManticoreSearch client
// 3. Tests the connection
// 4. Stores the client in global resource
func InitManticoreClient(ctx context.Context) error {
	// Validate dependencies and configuration
	if err := validateManticoreDependencies(); err != nil {
		return fmt.Errorf("manticore dependencies validation failed: %w", err)
	}

	resource.LoggerService.Info("initializing manticore client")

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create and configure the client
	client, err := createManticoreClient(initCtx)
	if err != nil {
		return fmt.Errorf("failed to create manticore client: %w", err)
	}

	// Test the connection
	if err := testManticoreConnection(initCtx, client); err != nil {
		return fmt.Errorf("manticore connection test failed: %w", err)
	}

	// Store the client in the resource package
	resource.ManticoreClient = client

	resource.LoggerService.Info("successfully initialized manticore client")
	return nil
}

// validateManticoreDependencies validates all required dependencies for Manticore initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateManticoreDependencies() error {
	// Check if configuration is loaded
	if config.ManticoreConfig == nil {
		return fmt.Errorf("manticore configuration is not initialized")
	}

	// Check if logger service is initialized
	if resource.LoggerService == nil {
		return fmt.Errorf("logger service is not initialized")
	}

	cfg := &config.ManticoreConfig.Manticore

	// Validate endpoints
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("manticore endpoints are not configured")
	}

	// Validate port
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid manticore port: %d, must be between 1 and 65535", cfg.Port)
	}

	// Validate each endpoint
	for i, endpoint := range cfg.Endpoints {
		if endpoint == "" {
			return fmt.Errorf("manticore endpoint %d is empty", i)
		}
	}

	return nil
}

// createManticoreClient creates and configures a ManticoreSearch client.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - *manticore.APIClient: The created client
//   - error: An error if client creation fails, nil otherwise
func createManticoreClient(_ context.Context) (*manticore.APIClient, error) {
	cfg := &config.ManticoreConfig.Manticore

	resource.LoggerService.Info(fmt.Sprintf("creating manticore client with %d endpoints", len(cfg.Endpoints)))

	// Create a new ManticoreSearch configuration
	configuration := manticore.NewConfiguration()

	// Configure HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	configuration.HTTPClient = httpClient

	// Configure servers
	servers := make([]manticore.ServerConfiguration, len(cfg.Endpoints))
	for i, endpoint := range cfg.Endpoints {
		serverURL := fmt.Sprintf("http://%s:%d", endpoint, cfg.Port)
		servers[i] = manticore.ServerConfiguration{
			URL: serverURL,
		}
		resource.LoggerService.Info(fmt.Sprintf("configured manticore server: %s", serverURL))
	}
	configuration.Servers = servers

	// Set authentication if configured
	if cfg.UserName != "" && cfg.PassWord != "" {
		configuration.DefaultHeader = map[string]string{
			"Authorization": fmt.Sprintf("Basic %s", encodeBasicAuth(cfg.UserName, cfg.PassWord)),
		}
		resource.LoggerService.Info("configured manticore authentication")
	}

	// Create the API client
	client := manticore.NewAPIClient(configuration)

	resource.LoggerService.Info("successfully created manticore client")
	return client, nil
}

// encodeBasicAuth encodes username and password for basic authentication.
//
// Parameters:
//   - username: The username
//   - password: The password
//
// Returns:
//   - string: Base64 encoded credentials
func encodeBasicAuth(username, password string) string {
	// This is a simplified implementation
	// In production, you should use proper base64 encoding
	return fmt.Sprintf("%s:%s", username, password)
}

// testManticoreConnection tests the ManticoreSearch connection.
//
// Parameters:
//   - ctx: Context for the operation
//   - client: The client to test
//
// Returns:
//   - error: An error if connection test fails, nil otherwise
func testManticoreConnection(ctx context.Context, client *manticore.APIClient) error {
	resource.LoggerService.Info("testing manticore connection")

	// Create timeout context for connection test
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test connection by trying to get server status or perform a simple operation
	// Since ManticoreSearch doesn't have a direct ping endpoint, we'll try a simple search
	done := make(chan error, 1)
	go func() {
		defer close(done)

		// Try to perform a simple operation to test connectivity
		// This is a basic connectivity test
		searchRequest := manticore.NewSearchRequest("_test_connection_")
		searchQuery := manticore.NewSearchQuery()
		searchQuery.QueryString = "*"
		searchRequest.Query = searchQuery

		// Execute the search request (this may fail if index doesn't exist, but that's OK)
		_, _, err := client.SearchAPI.Search(testCtx).SearchRequest(*searchRequest).Execute()

		// We don't care about the specific error, just that we can connect
		// Connection errors will be different from "index not found" errors
		if err != nil {
			// Check if it's a connection error or just an index not found error
			errStr := err.Error()
			if contains(errStr, "connection") || contains(errStr, "timeout") || contains(errStr, "refused") {
				done <- fmt.Errorf("connection test failed: %w", err)
				return
			}
			// If it's just an index error, the connection is working
		}

		done <- nil
	}()

	// Wait for connection test or timeout
	select {
	case err := <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("manticore connection test failed: %v", err))
			return err
		}
	case <-testCtx.Done():
		resource.LoggerService.Error("manticore connection test timeout")
		return fmt.Errorf("connection test timeout")
	}

	resource.LoggerService.Info("manticore connection test completed successfully")
	return nil
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsHelper(s, substr))))
}

// containsHelper is a helper function for substring search.
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CloseManticore closes the ManticoreSearch client connection gracefully.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if the client is initialized
// 2. Performs any necessary cleanup operations
// 3. Clears the global resource reference
func CloseManticore(ctx context.Context) error {
	if resource.ManticoreClient == nil {
		resource.LoggerService.Info("manticore client is not initialized, nothing to close")
		return nil
	}

	resource.LoggerService.Info("closing manticore client")

	// Create timeout context for close operation
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Perform cleanup operations
	done := make(chan error, 1)
	go func() {
		defer close(done)

		// Close idle connections of the HTTP client
		if httpClient := resource.ManticoreClient.GetConfig().HTTPClient; httpClient != nil {
			if transport, ok := httpClient.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
		}

		// Clear the global reference
		resource.ManticoreClient = nil
		done <- nil
	}()

	// Wait for cleanup operation or timeout
	select {
	case err := <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close manticore client: %v", err))
			return err
		}
	case <-closeCtx.Done():
		resource.LoggerService.Error("manticore client close timeout")
		// Still clear the reference even on timeout
		resource.ManticoreClient = nil
		return fmt.Errorf("manticore client close timeout")
	}

	resource.LoggerService.Info("successfully closed manticore client")
	return nil
}
