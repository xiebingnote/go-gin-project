package service

import (
	"context"
	"net/http"
	"time"

	"project/library/config"
	"project/library/resource"

	"github.com/olivere/elastic/v7"
)

// InitES initializes the Elasticsearch (ES) client using the configuration
// specified in the./conf/es.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ES cluster.
//
// The initialized ES client is stored as a singleton in the resource package
// for use throughout the application.
// If the configuration file decoding fails, the function panics with an error.
//
// The function takes a context.Context parameter, but does not currently use it.
func InitES(_ context.Context) {

	// Initialize the ES client with the decoded configuration.
	resource.ESClient = InitESClient()
}

// InitESClient initializes a new ES client with connection pool.
//
// The transport is set up with:
//
// - MaxIdleConns: config.ESConfig.ES.MaxIdleConns
// - MaxIdleConnsPerHost: config.ESConfig.ES.MaxIdleConnsPerHost
// - IdleConnTimeout: config.ESConfig.ES.IdleConnTimeout * time.Second
//
// The client is set up with:
//
// - SetURL: config.ESConfig.ES.Address
// - SetBasicAuth: config.ESConfig.ES.Username, config.ESConfig.ES.Password
// - SetHttpClient: the transport above
// - SetSniff: false
// - SetHealthcheck: false
//
// If the client initialization fails, the function will panic with the error.
func InitESClient() *elastic.Client {
	// Set the maximum number of idle (keep-alive) connections across all hosts.
	httpTransport := &http.Transport{
		MaxIdleConns: config.ESConfig.ES.MaxIdleConns,
	}

	// Set the maximum number of idle (keep-alive) connections per-host.
	httpTransport.MaxIdleConnsPerHost = config.ESConfig.ES.MaxIdleConnsPerHost

	// Set the time for which to keep an idle connection open waiting for a request.
	httpTransport.IdleConnTimeout = time.Duration(config.ESConfig.ES.IdleConnTimeout) * time.Second

	// Create an HTTP client with the above transport.
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	// Create a new ES client with the above client.
	client, err := elastic.NewClient(
		// Set the Elasticsearch URL to use.
		elastic.SetURL(config.ESConfig.ES.Address...),

		// Set the basic authentication username and password to use when connecting to Elasticsearch.
		elastic.SetBasicAuth(config.ESConfig.ES.Username, config.ESConfig.ES.Password),

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
