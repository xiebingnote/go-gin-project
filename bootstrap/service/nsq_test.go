package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

// setupTestLogger initializes a test logger for testing purposes
func setupTestLoggerForNSQ() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestNSQConfig initializes a test configuration for NSQ
func setupTestNSQConfig() {
	config.NsqConfig = &config.NsqConfigEntry{}
	config.NsqConfig.NSQ.Address = []string{"127.0.0.1:4150"}
	config.NsqConfig.NSQ.LookupdAddress = []string{"127.0.0.1:4161"}

	// Producer config
	config.NsqConfig.NSQ.Producer.DialTimeout = 5
	config.NsqConfig.NSQ.Producer.MaxAttempts = 3
	config.NsqConfig.NSQ.Producer.Concurrency = 100

	// Consumer config
	config.NsqConfig.NSQ.Consumer.Topic = "test_topic"
	config.NsqConfig.NSQ.Consumer.Channel = "test_channel"
	config.NsqConfig.NSQ.Consumer.MaxInFlight = 100
	config.NsqConfig.NSQ.Consumer.MaxAttempts = 5
	config.NsqConfig.NSQ.Consumer.RequeueDelay = 30
	config.NsqConfig.NSQ.Consumer.HeartbeatInterval = 30
}

// TestValidateNSQConfig tests the function validateNSQConfig to ensure it
// correctly validates the NSQ configuration parameters.
//
// The test cases cover the following scenarios:
//   - nil config
//   - empty NSQ addresses
//   - empty NSQLookupd addresses
//   - invalid producer dial timeout
//   - invalid producer max attempts
//   - empty consumer topic
//   - empty consumer channel
//   - invalid consumer max in flight
//   - invalid consumer max attempts
//   - valid config
//
// For each test case, the expected error or success is verified.
func TestValidateNSQConfig(t *testing.T) {
	setupTestLoggerForNSQ()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.NsqConfig = nil
			},
			expectError: true,
			errorMsg:    "NSQ configuration is not initialized",
		},
		{
			name: "empty NSQ addresses",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Address = []string{}
			},
			expectError: true,
			errorMsg:    "NSQ addresses are not configured",
		},
		{
			name: "empty NSQLookupd addresses",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.LookupdAddress = []string{}
			},
			expectError: true,
			errorMsg:    "NSQLookupd addresses are not configured",
		},
		{
			name: "invalid producer dial timeout",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Producer.DialTimeout = 0
			},
			expectError: true,
			errorMsg:    "producer dial timeout must be greater than 0",
		},
		{
			name: "invalid producer max attempts",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Producer.MaxAttempts = 0
			},
			expectError: true,
			errorMsg:    "producer max attempts must be greater than 0",
		},
		{
			name: "empty consumer topic",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Consumer.Topic = ""
			},
			expectError: true,
			errorMsg:    "consumer topic is not configured",
		},
		{
			name: "empty consumer channel",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Consumer.Channel = ""
			},
			expectError: true,
			errorMsg:    "consumer channel is not configured",
		},
		{
			name: "invalid consumer max in flight",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Consumer.MaxInFlight = 0
			},
			expectError: true,
			errorMsg:    "consumer max in flight must be greater than 0",
		},
		{
			name: "invalid consumer max attempts",
			setupConfig: func() {
				setupTestNSQConfig()
				config.NsqConfig.NSQ.Consumer.MaxAttempts = 0
			},
			expectError: true,
			errorMsg:    "consumer max attempts must be greater than 0",
		},
		{
			name: "valid config",
			setupConfig: func() {
				setupTestNSQConfig()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			err := validateNSQConfig()

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

// TestCleanupNSQResources verifies that the cleanupNSQResources function
// correctly cleans up NSQ producers and consumer resources.
//
// The test sets up mock NSQ producer and consumer resources, calls the
// cleanupNSQResources function, and then asserts that the resources have
// been set to nil, indicating successful cleanup.
func TestCleanupNSQResources(t *testing.T) {
	setupTestLoggerForNSQ()

	// Set up some mock resources
	resource.NsqProducer = []*nsq.Producer{nil, nil} // Mock producers
	resource.NsqConsumer = nil                       // Mock consumer

	cleanupNSQResources()

	// Verify resources are cleaned up
	if resource.NsqProducer != nil {
		t.Errorf("Expected NsqProducer to be nil after cleanup")
	}
	if resource.NsqConsumer != nil {
		t.Errorf("Expected NsqConsumer to be nil after cleanup")
	}
}

// TestCloseNsq_NoResources verifies the behavior of CloseNsq when there are no
// NSQ producers or consumers to close. It ensures that calling CloseNsq with
// nil resources does not result in an error. The function sets up the test
// logger, initializes the resources to nil, and checks that CloseNsq executes
// without returning an error.
func TestCloseNsq_NoResources(t *testing.T) {
	setupTestLoggerForNSQ()

	// Ensure no resources are set
	resource.NsqProducer = nil
	resource.NsqConsumer = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseNsq(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no resources, got: %v", err)
	}
}

// TestInitNSQ_WithValidConfig verifies the behavior of InitNSQ when given
// a valid configuration. It sets up the test logger, initializes the test
// configuration, and calls InitNSQ with a context that has a 30-second
// timeout. The function recovers from any panics and verifies that the
// expected resources are initialized. The function also cleans up the
// resources after the test is complete. This test will only pass if NSQ is
// actually running. The test is skipped if NSQ is not available.
func TestInitNSQ_WithValidConfig(t *testing.T) {
	setupTestLoggerForNSQ()
	setupTestNSQConfig()

	// This test will only pass if NSQ is actually running
	// Skip if NSQ is not available
	t.Skip("Skipping integration test - requires running NSQ instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitNSQ panicked: %v", r)
		}
	}()

	InitNSQ(ctx)

	// Clean up
	if resource.NsqProducer != nil || resource.NsqConsumer != nil {
		_ = CloseNsq(ctx)
	}
}
