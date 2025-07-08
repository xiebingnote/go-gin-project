package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// InitMongoDB initializes the MongoDB database connection.
//
// This function calls InitMongoDBClient to establish a connection to the MongoDB
// database using the configuration provided.
//
// If the connection cannot be established, it panics with an error message.
func InitMongoDB(ctx context.Context) {
	// Attempt to initialize the MongoDB client with context
	if err := InitMongoDBClient(ctx); err != nil {
		// Log the error before panicking
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize MongoDB: %v", err))
		panic(fmt.Sprintf("MongoDB initialization failed: %v", err))
	}
}

// InitMongoDBClient initializes a new MongoDB client with connection pool.
//
// The MongoDB client is initialized by constructing a connection URI from the
// configuration parameters in the global MongoConfigEntry. The client is then
// connected to the MongoDB database using the client.Connect method.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Constructs the MongoDB connection URI based on authentication settings
// 2. Configures client options including connection pool settings
// 3. Establishes connection to MongoDB server
// 4. Tests the connection by pinging the server
// 5. Stores the initialized client in the global resource package
func InitMongoDBClient(ctx context.Context) error {
	// Validate configuration
	if config.MongoConfig == nil {
		return fmt.Errorf("MongoDB configuration is not initialized")
	}

	cfg := &config.MongoConfig.Mongo
	if cfg.Host == "" {
		return fmt.Errorf("MongoDB host is not configured")
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("MongoDB port is not configured or invalid")
	}
	if cfg.DBName == "" {
		return fmt.Errorf("MongoDB database name is not configured")
	}

	// Construct the MongoDB connection URI
	var uri string
	if cfg.Username == "" {
		// If the username and password are not set, use the host and port only
		uri = fmt.Sprintf("mongodb://%s:%v", cfg.Host, cfg.Port)
		resource.LoggerService.Info("Connecting to MongoDB without authentication")
	} else {
		// If the username and password are set, use them in the URI
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%v", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
		resource.LoggerService.Info(fmt.Sprintf("Connecting to MongoDB with authentication for user: %s", cfg.Username))
	}

	// Create a new ClientOptions object and apply the URI to it
	clientOptions := options.Client().ApplyURI(uri)

	// Configure connection pool settings
	if cfg.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(cfg.MinPoolSize)
	}
	if cfg.MaxConnIdleTime > 0 {
		clientOptions.SetMaxConnIdleTime(cfg.MaxConnIdleTime * time.Millisecond)
	}

	// Configure timeout settings
	if cfg.ConnectTimeout > 0 {
		clientOptions.SetConnectTimeout(cfg.ConnectTimeout * time.Millisecond)
	}
	if cfg.ServerTimeout > 0 {
		clientOptions.SetServerSelectionTimeout(cfg.ServerTimeout * time.Millisecond)
	}

	// Connect to MongoDB using the clientOptions with provided context
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		// If the connection fails, return an error
		resource.LoggerService.Error(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ensure client is disconnected if subsequent operations fail
	defer func() {
		if err != nil {
			if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
				resource.LoggerService.Error(fmt.Sprintf("Failed to disconnect MongoDB client during cleanup: %v", disconnectErr))
			}
		}
	}()

	// Test the connection by pinging the MongoDB server with primary read preference
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = client.Ping(pingCtx, readpref.Primary())
	if err != nil {
		// If the ping fails, return an error
		resource.LoggerService.Error(fmt.Sprintf("Failed to ping MongoDB server: %v", err))
		return fmt.Errorf("failed to ping MongoDB server: %w", err)
	}

	// Get database instance
	database := client.Database(cfg.DBName)
	if database == nil {
		resource.LoggerService.Error("Failed to get MongoDB database instance")
		return fmt.Errorf("failed to get MongoDB database instance")
	}

	// Store the initialized MongoDB client in the resource package
	resource.MongoDBClient = database

	// Log successful connection with connection details
	resource.LoggerService.Info(fmt.Sprintf("✅ successfully connected to MongoDB database '%s' at %s:%v",
		cfg.DBName, cfg.Host, cfg.Port))

	// Return nil to indicate successful initialization
	return nil
}

// CloseMongoDB closes the MongoDB client connection gracefully.
//
// This function checks if the global MongoDBClient resource is initialized.
// If it is, it attempts to disconnect the client connection with a timeout
// and returns an error if the disconnection fails.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client disconnect operation fails, nil otherwise
func CloseMongoDB(ctx context.Context) error {
	// Check if the global MongoDBClient resource is initialized
	if resource.MongoDBClient == nil {
		return nil
	}

	// Create a timeout context for the disconnect operation
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Attempt to disconnect the MongoDB client connection
	if err := resource.MongoDBClient.Client().Disconnect(disconnectCtx); err != nil {
		// Log and return an error if disconnecting the connection fails
		resource.LoggerService.Error(fmt.Sprintf("Failed to close MongoDB client: %v", err))
		return fmt.Errorf("failed to close MongoDB client: %w", err)
	}

	// Clear the global reference
	resource.MongoDBClient = nil

	// Log successful disconnection (check if logger is still available)
	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 successfully disconnected from MongoDB")
	}
	return nil
}
