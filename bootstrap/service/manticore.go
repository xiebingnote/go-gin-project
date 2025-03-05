package service

import (
	"context"
	"fmt"

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
// If the configuration file decoding fails, the function panics with an error.
func InitManticore(_ context.Context) {
	if err := InitManticoreClient(); err != nil {
		// The ManticoreSearch client cannot be initialized. Panic with the error message.
		panic(err.Error())
	}
}

// InitManticoreClient initializes the ManticoreSearch client using the configuration
// specified in the ./conf/manticore.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ManticoreSearch cluster and creates a new ManticoreSearch APIClient.
//
// The initialized ManticoreSearch client is stored as a singleton in the
// resource package for use throughout the application.
//
// If the configuration file decoding fails, the function returns an error.
func InitManticoreClient() error {

	// Create a new ManticoreSearch configuration
	configuration := manticore.NewConfiguration()

	// Check if the ManticoreSearch host is empty
	if len(config.ManticoreConfig.Manticore.Endpoints) == 0 {
		resource.LoggerService.Error(fmt.Sprintf("manticore Host is empty"))
		return fmt.Errorf("manticore Host is empty")
	}

	// Set the ManticoreSearch server URLs
	for i := 0; i < len(config.ManticoreConfig.Manticore.Endpoints); i++ {
		configuration.Servers[i].URL = fmt.Sprintf("http://%s:%v", config.ManticoreConfig.Manticore.Endpoints[i], config.ManticoreConfig.Manticore.Port)
	}

	// Create a new ManticoreSearch APIClient
	resource.ManticoreClient = manticore.NewAPIClient(configuration)

	return nil
}

// CloseManticore closes the ManticoreSearch client connection.
//
// This function checks if the global ManticoreClient resource is initialized.
// If it is, it attempts to close idle connections associated with the HTTP client.
// If the Manticore client is nil, it indicates that there is no connection to close,
// and the function returns nil.
func CloseManticore() error {
	if resource.ManticoreClient != nil {
		// Attempt to close idle connections of the HTTP client
		resource.ManticoreClient.GetConfig().HTTPClient.CloseIdleConnections()
	}
	// Manticore client is nil, no connection to close
	return nil
}
