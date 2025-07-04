package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/olivere/elastic/v7"
)

// InitElasticSearch initializes the Elasticsearch client with the configuration
// specified in the ./conf/elasticsearch.toml file.
func InitElasticSearch(_ context.Context) {
	if err := InitElasticSearchClient(); err != nil {
		// Log an error message if the Elasticsearch connection cannot be established
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize Elasticsearch: %v", err))
		panic(err.Error())
	}
}

// InitElasticSearchClient initializes a new Elasticsearch client with connection pool.
//
// The function performs the following steps:
// 1. Validates the Elasticsearch configuration
// 2. Configures the HTTP transport
// 3. Creates an HTTP client
// 4. Initializes the Elasticsearch client
// 5. Tests the connection
// 6. Stores the initialized client in the global resource
//
// Returns an error if the configuration is invalid, the connection cannot be established,
// or any of the steps fail.
func InitElasticSearchClient() error {
	cfg := config.ElasticSearchConfig

	// Validate the Elasticsearch configuration
	if err := ValidateElasticSearchConfig(cfg); err != nil {
		return fmt.Errorf("invalid Elasticsearch configuration: %w", err)
	}

	// Configure the HTTP transport
	httpTransport := ConfigureElasticSearchTransport(cfg)

	// Create an HTTP client with the configured transport
	httpClient := &http.Client{
		Transport: httpTransport,
		Timeout:   30 * time.Second, // Set default timeout
	}

	// Initialize the Elasticsearch client
	client, err := elastic.NewClient(
		elastic.SetURL(cfg.ElasticSearch.Address...),
		elastic.SetBasicAuth(cfg.ElasticSearch.Username, cfg.ElasticSearch.Password),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetRetrier(elastic.NewBackoffRetrier(elastic.NewExponentialBackoff(100*time.Millisecond, 30*time.Second))),
		elastic.SetMaxRetries(3),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Elasticsearch client: %w", err)
	}

	// Test the connection
	if err := TestElasticSearchConnection(client); err != nil {
		return fmt.Errorf("failed to test Elasticsearch connection: %w", err)
	}

	// Store the initialized client in the global resource
	resource.ElasticSearchClient = client
	resource.LoggerService.Info("✅ successfully connected to Elasticsearch")

	return nil
}

// ValidateElasticSearchConfig validates the Elasticsearch configuration.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Address
//     - Username
//     - Password
//  3. Connection pool settings are valid:
//     - MaxIdleConns
//     - MaxIdleConnsPerHost
//     - IdleConnTimeout
//  4. Address format validation
func ValidateElasticSearchConfig(cfg *config.ElasticSearchConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("elasticsearch configuration is nil")
	}

	// Check required connection parameters
	if len(cfg.ElasticSearch.Address) == 0 {
		return fmt.Errorf("elasticsearch address list is empty")
	}

	// Validate each address
	for i, addr := range cfg.ElasticSearch.Address {
		if addr == "" {
			return fmt.Errorf("elasticsearch address[%d] is empty", i)
		}
		// Basic URL format validation
		if !isValidURL(addr) {
			return fmt.Errorf("invalid Elasticsearch address[%d]: %s", i, addr)
		}
	}

	if cfg.ElasticSearch.Username == "" {
		return fmt.Errorf("elasticsearch username is empty")
	}
	if cfg.ElasticSearch.Password == "" {
		return fmt.Errorf("elasticsearch password is empty")
	}

	// Check connection pool settings
	if cfg.ElasticSearch.MaxIdleConns <= 0 {
		return fmt.Errorf("invalid maximum idle connections: %d, must be greater than 0", cfg.ElasticSearch.MaxIdleConns)
	}
	if cfg.ElasticSearch.MaxIdleConnsPerHost <= 0 {
		return fmt.Errorf("invalid maximum idle connections per host: %d, must be greater than 0", cfg.ElasticSearch.MaxIdleConnsPerHost)
	}
	if cfg.ElasticSearch.MaxIdleConnsPerHost > cfg.ElasticSearch.MaxIdleConns {
		return fmt.Errorf("max idle connections per host (%d) cannot be greater than max idle connections (%d)",
			cfg.ElasticSearch.MaxIdleConnsPerHost, cfg.ElasticSearch.MaxIdleConns)
	}
	if cfg.ElasticSearch.IdleConnTimeout <= 0 {
		return fmt.Errorf("invalid idle connection timeout: %d seconds, must be greater than 0", cfg.ElasticSearch.IdleConnTimeout)
	}

	return nil
}

// ConfigureElasticSearchTransport configures the HTTP transport for Elasticsearch.
//
// Parameters:
//   - cfg: A pointer to the Elasticsearch configuration containing the connection
//     pool settings.
//
// Returns:
//   - A configured *http.Transport instance.
func ConfigureElasticSearchTransport(cfg *config.ElasticSearchConfigEntry) *http.Transport {
	return &http.Transport{
		MaxIdleConns:           cfg.ElasticSearch.MaxIdleConns,
		MaxIdleConnsPerHost:    cfg.ElasticSearch.MaxIdleConnsPerHost,
		IdleConnTimeout:        time.Duration(cfg.ElasticSearch.IdleConnTimeout) * time.Second,
		DisableKeepAlives:      false,
		DisableCompression:     false,
		MaxConnsPerHost:        cfg.ElasticSearch.MaxIdleConnsPerHost * 2,
		MaxResponseHeaderBytes: 4096,
		ResponseHeaderTimeout:  10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
	}
}

// TestElasticSearchConnection tests the Elasticsearch connection.
//
// The function takes an Elasticsearch client instance as a parameter and tests
// the connection by executing a simple ping request. If the request fails,
// the function returns an error.
//
// Parameters:
//   - client: An Elasticsearch client instance to test.
//
// Returns:
//   - An error if the ping request fails
//   - nil if the ping request succeeds
func TestElasticSearchConnection(client *elastic.Client) error {
	cfg := config.ElasticSearchConfig
	if cfg == nil || len(cfg.ElasticSearch.Address) == 0 {
		return fmt.Errorf("elasticsearch configuration or address is empty")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection using the first address
	_, code, err := client.Ping(cfg.ElasticSearch.Address[0]).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch at %s: %w", cfg.ElasticSearch.Address[0], err)
	}
	if code != 200 {
		return fmt.Errorf("unexpected status code from Elasticsearch at %s: %d", cfg.ElasticSearch.Address[0], code)
	}

	return nil
}

// CloseElasticSearch closes the Elasticsearch connection.
//
// This function attempts to close the global Elasticsearch client connection.
// If the connection is successfully closed, it returns nil. If the client is nil,
// it also returns nil.
//
// Returns:
//   - An error if there is an issue closing the connection
//   - nil if the Elasticsearch client is nil or the connection is closed successfully
func CloseElasticSearch() error {
	// Check if the global Elasticsearch client is initialized
	if resource.ElasticSearchClient == nil {
		// The Elasticsearch client is nil, no connection to close
		return nil
	}

	// Attempt to close the Elasticsearch connection
	resource.ElasticSearchClient.Stop()

	// Reset the global Elasticsearch client to nil
	resource.ElasticSearchClient = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("✅ successfully closed Elasticsearch connection")
	}

	// Return nil to indicate success
	return nil
}

// isValidURL validates if the given string is a valid URL format.
//
// Parameters:
//   - urlStr: The URL string to validate.
//
// Returns:
//   - true if the URL is valid, false otherwise.
func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
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
