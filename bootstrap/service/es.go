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
// specified in the ../conf/es.toml file. It reads the configuration parameters
// required to connect and authenticate with the ES cluster. The initialized ES
// client is stored as a singleton in the resource package for use throughout
// the application. If the configuration file decoding fails, the function
// panics with an error.
func InitES(_ context.Context) {

	// Initialize the ES client with the decoded configuration.
	resource.ESClient = InitESClient()
}

// InitESClient initializes a new ES client with connection pool.
//
// The transport is setup with:
//
// - MaxIdleConns: config.ESConfig.ES.MaxIdleConns
// - MaxIdleConnsPerHost: config.ESConfig.ES.MaxIdleConnsPerHost
// - IdleConnTimeout: config.ESConfig.ES.IdleConnTimeout * time.Second
//
// The client is setup with:
//
// - SetURL: config.ESConfig.ES.Address
// - SetBasicAuth: config.ESConfig.ES.Username, config.ESConfig.ES.Password
// - SetHttpClient: the transport above
// - SetSniff: false
// - SetHealthcheck: false
//
// If the client initialization fails, the function will panic with the error.
func InitESClient() *elastic.Client {
	// 使用连接池
	// Use connect pool.
	httpClient := &http.Client{
		Transport: &http.Transport{
			// MaxIdleConns: The maximum number of idle (keep-alive) connections across all hosts.
			MaxIdleConns: config.ESConfig.ES.MaxIdleConns,
			// MaxIdleConnsPerHost: The maximum number of idle (keep-alive) connections per-host.
			MaxIdleConnsPerHost: config.ESConfig.ES.MaxIdleConnsPerHost,
			// IdleConnTimeout: The time for which to keep an idle connection open waiting for a request.
			IdleConnTimeout: time.Duration(config.ESConfig.ES.IdleConnTimeout) * time.Second,
		},
	}

	client, err := elastic.NewClient(
		// SetURL: The Elasticsearch URL to use.
		elastic.SetURL(config.ESConfig.ES.Address...),
		// SetBasicAuth: The basic authentication username and password to use when connecting to Elasticsearch.
		elastic.SetBasicAuth(config.ESConfig.ES.Username, config.ESConfig.ES.Password),
		// SetHttpClient: The HTTP client to use when connecting to Elasticsearch.
		elastic.SetHttpClient(httpClient),
		// SetSniff: Whether or not to enable sniffing.
		elastic.SetSniff(false),
		// SetHealthcheck: Whether or not to enable health checking.
		elastic.SetHealthcheck(false),
	)

	// Panic if client initialization fails.
	if err != nil {
		panic(err.Error())
	}

	// Return the initialized Elasticsearch client.
	return client
}
