package service

import (
	"context"
	"fmt"
	"time"

	"project/library/config"
	"project/library/resource"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// InitEtcd initializes the Etcd client.
//
// This function is used to initialize the Etcd client by reading the Etcd
// configuration from the global EtcdConfig, and then create an Etcd client
// using the configuration.
//
// Finally, it assigns the client to the global EtcdClient resource.
//
// If any error occurs during the initialization of the Etcd client, the
// function will panic with the error message.
func InitEtcd(_ context.Context) {
	err := InitEtcdClient()
	if err != nil {
		panic(err.Error())
	}
}

// InitEtcdClient initializes the etcd client using the configuration specified
// in the global EtcdConfig.
//
// It sets up the client with the provided endpoints, dial timeout, and
// authentication details.
//
// The function also performs a health check to ensure the client can communicate
// with the etcd server.
//
// Returns an error if the client cannot be initialized or if the health check fails.
func InitEtcdClient() error {
	// Prepare the etcd client configuration
	clientConfig := clientv3.Config{
		Endpoints:   config.EtcdConfig.Etcd.Endpoints,                 // etcd server endpoints
		DialTimeout: config.EtcdConfig.Etcd.DialTimeout * time.Second, // dial timeout duration
		Username:    config.EtcdConfig.Etcd.Username,                  // authentication username
		Password:    config.EtcdConfig.Etcd.Password,                  // authentication password
	}

	// Create the etcd client
	cli, err := clientv3.New(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	// Perform a connection health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err = cli.Status(ctx, clientConfig.Endpoints[0]); err != nil {
		return fmt.Errorf("etcd health check failed: %w", err)
	}

	// Log successful initialization
	resource.LoggerService.Info("Etcd client initialized successfully")

	return nil
}

// CloseEtcd safely closes the etcd client connection.
//
// This function checks if the global EtcdClient resource is initialized.
// If it is, it attempts to close the client connection.
//
// It logs an error message if the closure fails and returns the error.
//
// If successful, it logs an informational message indicating the client
// was closed.
//
// Returns an error if the client close operation fails.
func CloseEtcd() error {
	if resource.EtcdClient != nil {
		// Attempt to close the etcd client connection
		if err := resource.EtcdClient.Close(); err != nil {
			// Log an error if the closure fails
			resource.LoggerService.Error(fmt.Sprintf("Etcd client close failed, err: %v", err))
			return err
		}
		// Log an informational message if the closure is successful
		resource.LoggerService.Info("Etcd client closed successfully")
	}
	return nil
}
