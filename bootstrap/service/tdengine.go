package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	//_ "github.com/taosdata/driver-go/v3/taosSql"
)

// InitTDengine initializes the TDengine database connection.
//
// This function calls InitTDengineClient to establish a connection to the TDengine
// database using the configuration provided.
//
// If the connection cannot be established, it panics with an error message.
func InitTDengine(_ context.Context) {
	if err := InitTDengineClient(); err != nil {
		// The TDengine client cannot be initialized. Panic with the error message.
		panic(err)
	}
}

// InitTDengineClient initializes the TDengine client with the configuration
// specified in the ./conf/tdengine.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the TDengine database.
//
// The initialized TDengine client is stored as a singleton in the
// resource package for use throughout the application.
//
// If the connection cannot be established, it panics with an error message.
func InitTDengineClient() error {
	// Create the DSN (Data Source Name) for the TDengine client.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%v)/%s", config.TDengineConfig.TDengine.UserName, config.TDengineConfig.TDengine.PassWord,
		config.TDengineConfig.TDengine.Host, config.TDengineConfig.TDengine.Port, config.TDengineConfig.TDengine.Database)

	// Open a connection to the TDengine database using the DSN.
	db, err := sql.Open("taosSql", dsn)
	if err != nil {
		// The TDengine client cannot be initialized. Log the error message.
		resource.LoggerService.Error(fmt.Sprintf("open tdengine client failed: %v", err))
		return err
	}

	// Assign the initialized TDengine client to the global TDengine client resource
	// This is used by the application to interact with the TDengine database.
	resource.TDengineClient = db

	return nil
}

// CloseTDengine closes the TDengine database connection.
//
// It checks if the global TDengine client resource is initialized.
// If it is, it attempts to close the client connection and returns an error
// if the closure fails. If successful, it returns nil.
//
// Returns:
//   - An error if the client close operation fails.
//   - nil if the TDengine client is nil or the connection is closed successfully.
func CloseTDengine() error {
	if resource.TDengineClient != nil {
		if err := resource.TDengineClient.Close(); err != nil {
			// The TDengine client cannot be closed. Log the error message.
			resource.LoggerService.Error(fmt.Sprintf("close tdengine client failed: %v", err))
			return err
		}
	}

	// TDengine client is nil, no connection to close
	return nil
}
