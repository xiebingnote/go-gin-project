package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		client.GetConfig().HTTPClient.CloseIdleConnections()
		return fmt.Errorf("manticore connection test failed: %w", err)
	}

	// Store the client in the resource package
	resource.ManticoreClient = client

	resource.LoggerService.Info("✅ successfully initialized manticore client")
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
func createManticoreClient(ctx context.Context) (*manticore.APIClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg := &config.ManticoreConfig.Manticore
	configuration := manticore.NewConfiguration()
	configuration.Servers = nil
	for _, endpoint := range cfg.Endpoints {
		if !strings.Contains(endpoint, "://") {
			endpoint = "http://" + net.JoinHostPort(endpoint, strconv.Itoa(cfg.Port))
		}
		u, err := url.Parse(endpoint)
		if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("invalid manticore endpoint (use a host or an HTTP(S) URL without credentials or query parameters)")
		}
		if cfg.UserName != "" || cfg.PassWord != "" {
			if cfg.UserName == "" || strings.Contains(cfg.UserName, ":") {
				return nil, fmt.Errorf("invalid manticore basic auth username")
			}
			if u.Scheme != "https" {
				return nil, fmt.Errorf("manticore authentication requires HTTPS endpoints")
			}
		}
		configuration.Servers = append(configuration.Servers, manticore.ServerConfiguration{URL: strings.TrimRight(u.String(), "/")})
	}
	configuration.HTTPClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{MaxIdleConns: 10, MaxIdleConnsPerHost: 10, IdleConnTimeout: 30 * time.Second},
		// Keep credentials on the configured server and prohibit HTTPS downgrades.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	if cfg.UserName != "" {
		configuration.DefaultHeader["Authorization"] = "Basic " + encodeBasicAuth(cfg.UserName, cfg.PassWord)
	}
	return manticore.NewAPIClient(configuration), nil
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
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
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
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// SELECT 1 does not require an application index and does not modify data.
	result, response, err := client.UtilsAPI.Sql(testCtx).Body("SELECT 1").Execute()
	if err != nil {
		return fmt.Errorf("manticore health check failed: %w", err)
	}
	if response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("manticore health check returned an unsuccessful HTTP status")
	}
	if result == nil {
		return fmt.Errorf("manticore health check returned no result")
	}
	var rows []map[string]interface{}
	if result.ArrayOfMapmapOfStringinterface != nil {
		rows = *result.ArrayOfMapmapOfStringinterface
	}
	if result.MapmapOfStringinterface != nil {
		rows = append(rows, *result.MapmapOfStringinterface)
	}
	if len(rows) == 0 {
		return fmt.Errorf("manticore health check returned an empty result")
	}
	for _, row := range rows {
		// SQL failures can also be returned with HTTP 200.
		if queryErr, ok := row["error"]; ok && queryErr != nil && fmt.Sprint(queryErr) != "" {
			return fmt.Errorf("manticore health query failed: %v", queryErr)
		}
	}
	return nil
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
	client := resource.ManticoreClient
	if client == nil {
		return nil
	}
	httpClient := client.GetConfig().HTTPClient
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := waitForClose(closeCtx, func() error {
		if httpClient != nil {
			httpClient.CloseIdleConnections()
		}
		return nil
	}); err != nil {
		return fmt.Errorf("close manticore: %w", err)
	}
	resource.ManticoreClient = nil
	return nil
}
