package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLogger initializes a test logger for testing purposes
func setupTestLogger() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestConfig initializes a test configuration for MongoDB
func setupTestConfig() {
	config.MongoConfig = &config.MongoDBConfigEntry{}
	config.MongoConfig.Mongo.Host = "localhost"
	config.MongoConfig.Mongo.Port = 27017
	config.MongoConfig.Mongo.DBName = "test"
	config.MongoConfig.Mongo.ConnectTimeout = 10000
	config.MongoConfig.Mongo.ServerTimeout = 30000
	config.MongoConfig.Mongo.MaxPoolSize = 100
	config.MongoConfig.Mongo.MinPoolSize = 0
	config.MongoConfig.Mongo.MaxConnIdleTime = 0
}

func TestInitMongoDBClient_ConfigValidation(t *testing.T) {
	setupTestLogger()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.MongoConfig = nil
			},
			expectError: true,
			errorMsg:    "MongoDB configuration is not initialized",
		},
		{
			name: "empty host",
			setupConfig: func() {
				setupTestConfig()
				config.MongoConfig.Mongo.Host = ""
			},
			expectError: true,
			errorMsg:    "MongoDB host is not configured",
		},
		{
			name: "invalid port",
			setupConfig: func() {
				setupTestConfig()
				config.MongoConfig.Mongo.Port = 0
			},
			expectError: true,
			errorMsg:    "MongoDB port is not configured or invalid",
		},
		{
			name: "empty database name",
			setupConfig: func() {
				setupTestConfig()
				config.MongoConfig.Mongo.DBName = ""
			},
			expectError: true,
			errorMsg:    "MongoDB database name is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := InitMongoDBClient(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCloseMongoDB_NilClient(t *testing.T) {
	setupTestLogger()

	// Ensure client is nil
	resource.MongoDBClient = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseMongoDB(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing nil client, got: %v", err)
	}
}

func TestInitMongoDB_WithValidConfig(t *testing.T) {
	setupTestLogger()
	setupTestConfig()

	// This test will only pass if MongoDB is actually running
	// Skip if MongoDB is not available
	t.Skip("Skipping integration test - requires running MongoDB instance")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitMongoDB panicked: %v", r)
		}
	}()

	InitMongoDB(ctx)

	// Clean up
	if resource.MongoDBClient != nil {
		_ = CloseMongoDB(ctx)
	}
}
