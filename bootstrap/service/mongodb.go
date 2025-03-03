package service

import (
	"context"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitMongoDB initializes the MongoDB database connection.
//
// This function calls InitMongoDBClient to establish a connection to the MongoDB
// database using the configuration provided.
//
// If the connection cannot be established, it panics with an error message.
func InitMongoDB(_ context.Context) {
	// Attempt to initialize the MongoDB client
	if err := InitMongoDBClient(); err != nil {
		// Panic if the MongoDB client cannot be initialized
		panic(err.Error())
	}
}

// InitMongoDBClient initializes a new MongoDB client with connection pool.
//
// The MongoDB client is initialized by constructing a connection URI from the
// configuration parameters in the global MongoConfigEntry. The client is then
// connected to the MongoDB database using the client.Connect method.
//
// If the client initialization fails, the function will return an error with a
// formatted message indicating the reason for the failure.
//
// The function also checks the connection to MongoDB by pinging the server. If the
// ping fails, the function will return an error.
//
// If the connection is successful, the function logs a message to indicate a
// successful connection to MongoDB and stores the initialized MongoDB client in
// the global resource package.
func InitMongoDBClient() error {
	// Construct the MongoDB connection URI
	var uri string
	switch config.MongoConfig.Mongo.Username == "" {
	case true:
		// If the username and password are not set, use the host and port only.
		uri = fmt.Sprintf("mongodb://%s:%v", config.MongoConfig.Mongo.Host, config.MongoConfig.Mongo.Port)
	default:
		// If the username and password are set, use them in the URI.
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%v", config.MongoConfig.Mongo.Username, config.MongoConfig.Mongo.Password,
			config.MongoConfig.Mongo.Host, config.MongoConfig.Mongo.Port)
	}

	// Create a new ClientOptions object and apply the URI to it.
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to MongoDB using the clientOptions.
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		// If the connection fails, return an error.
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the MongoDB server to check the connection.
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		// If the ping fails, return an error.
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Log a message to indicate a successful connection to MongoDB.
	resource.LoggerService.Info("Successfully connected to MongoDB!")

	// Store the initialized MongoDB client in the resource package.
	// The MongoDB client is a pointer to a GORM database connection.
	resource.MongoDBClient = client.Database(config.MongoConfig.Mongo.DBName)

	// Return nil to indicate successful initialization.
	return nil
}

// CloseMongoDB closes the MongoDB client connection.
//
// This function checks if the global MongoDBClient resource is initialized.
// If it is, it attempts to disconnect the client connection and returns an error
// if the disconnection fails. If successful, it returns nil.
//
// Returns:
//   - An error if the client disconnect operation fails.
//   - nil if the MongoDB client is nil or the connection is closed successfully.
func CloseMongoDB() error {
	// Check if the global MongoDBClient resource is initialized.
	if resource.MongoDBClient != nil {
		// Attempt to disconnect the MongoDB client connection.
		if err := resource.MongoDBClient.Client().Disconnect(context.Background()); err != nil {
			// Return an error if disconnecting the connection fails.
			return fmt.Errorf("failed to close MongoDB Client: %w", err)
		}
	}
	// MongoDB client is nil, no connection to close.
	return nil
}
