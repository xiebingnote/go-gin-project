package service

import (
	"context"
	"net/http"
	"time"

	"go-gin-project/library/config"
	"go-gin-project/library/resource"

	"github.com/olivere/elastic/v7"
)

// InitElasticSearch initializes the Elasticsearch (ElasticSearch) client using the configuration
// specified in the./conf/elasticsearch.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ElasticSearch cluster.
//
// The initialized ElasticSearch client is stored as a singleton in the resource package
// for use throughout the application.
// If the configuration file decoding fails, the function panics with an error.
//
// The function takes a context.Context parameter, but does not currently use it.
func InitElasticSearch(_ context.Context) {

	// Initialize the ElasticSearch client with the decoded configuration.
	resource.ElasticSearchClient = InitElasticSearchClient()
}

// InitElasticSearchClient initializes a new ElasticSearch client with connection pool.
//
// The transport is set up with:
//
// - MaxIdleConns: config.ElasticSearchConfig.ElasticSearch.MaxIdleConns
// - MaxIdleConnsPerHost: config.ElasticSearchConfig.ElasticSearch.MaxIdleConnsPerHost
// - IdleConnTimeout: config.ElasticSearchConfig.ElasticSearch.IdleConnTimeout * time.Second
//
// The client is set up with:
//
// - SetURL: config.ElasticSearchConfig.ElasticSearch.Address
// - SetBasicAuth: config.ElasticSearchConfig.ElasticSearch.Username, config.ElasticSearchConfig.ElasticSearch.Password
// - SetHttpClient: the transport above
// - SetSniff: false
// - SetHealthcheck: false
//
// If the client initialization fails, the function will panic with the error.
func InitElasticSearchClient() *elastic.Client {
	// Set the maximum number of idle (keep-alive) connections across all hosts.
	httpTransport := &http.Transport{
		MaxIdleConns: config.ElasticSearchConfig.ElasticSearch.MaxIdleConns,
	}

	// Set the maximum number of idle (keep-alive) connections per-host.
	httpTransport.MaxIdleConnsPerHost = config.ElasticSearchConfig.ElasticSearch.MaxIdleConnsPerHost

	// Set the time for which to keep an idle connection open waiting for a request.
	httpTransport.IdleConnTimeout = time.Duration(config.ElasticSearchConfig.ElasticSearch.IdleConnTimeout) * time.Second

	// Create an HTTP client with the above transport.
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	// Create a new ElasticSearch client with the above client.
	client, err := elastic.NewClient(
		// Set the Elasticsearch URL to use.
		elastic.SetURL(config.ElasticSearchConfig.ElasticSearch.Address...),

		// Set the basic authentication username and password to use when connecting to Elasticsearch.
		elastic.SetBasicAuth(config.ElasticSearchConfig.ElasticSearch.Username, config.ElasticSearchConfig.ElasticSearch.Password),

		// Set the HTTP client to use when connecting to Elasticsearch.
		elastic.SetHttpClient(httpClient),

		// Set whether to enable sniffing.
		elastic.SetSniff(false),

		// Set whether to enable health checking.
		elastic.SetHealthcheck(false),
	)

	// Panic if client initialization fails.
	if err != nil {
		panic(err.Error())
	}

	// Return the initialized Elasticsearch client.
	return client
}

// CloseElasticSearch closes the ElasticSearch client connection.
//
// It checks if the global ElasticSearch client resource is initialized.
// If it is, it attempts to close the client connection and returns an error
// if the closure fails.
// If successful, it returns nil.
//
// Returns:
//   - An error if the client close operation fails.
//   - nil if the ElasticSearch client is nil or the connection is closed successfully.
func CloseElasticSearch() error {
	// Check if the global ElasticSearch client resource is initialized.
	if resource.ElasticSearchClient != nil {
		// Attempt to close the ElasticSearch client connection
		resource.ElasticSearchClient.Stop()
		return nil
	}
	// ElasticSearch client is nil, no connection to close
	return nil
}
