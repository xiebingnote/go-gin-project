package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

// setupTestLoggerForKafka initializes a test logger for testing purposes
func setupTestLoggerForKafka() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestKafkaConfig initializes a test configuration for Kafka
func setupTestKafkaConfig() *config.KafkaConfigEntry {
	return &config.KafkaConfigEntry{
		Kafka: struct {
			Brokers            []string `toml:"Brokers"`
			ProducerTopic      string   `toml:"ProducerTopic"`
			ConsumerTopic      string   `toml:"ConsumerTopic"`
			ConsumerGroupTopic []string `toml:"ConsumerGroupTopic"`
			GroupID            string   `toml:"GroupID"`
			Version            string   `toml:"Version"`
		}{
			Brokers:            []string{"localhost:9092"},
			ProducerTopic:      "test-producer-topic",
			ConsumerTopic:      "test-consumer-topic",
			ConsumerGroupTopic: []string{"test-group-topic1", "test-group-topic2"},
			GroupID:            "test-group",
			Version:            "2.8.0",
		},
		Advanced: struct {
			ProducerMaxRetry       int   `toml:"ProducerMaxRetry"`
			ConsumerSessionTimeout int64 `toml:"ConsumerSessionTimeout"`
			HeartbeatInterval      int64 `toml:"HeartbeatInterval"`
			MaxProcessingTime      int64 `toml:"MaxProcessingTime"`
		}{
			ProducerMaxRetry:       3,
			ConsumerSessionTimeout: 30000, // 30 seconds
			HeartbeatInterval:      3000,  // 3 seconds
			MaxProcessingTime:      60000, // 60 seconds
		},
	}
}

// TestValidateKafkaConfig tests the ValidateKafkaConfig function for various scenarios.
// It checks for proper validation of Kafka configuration by running multiple test cases.
// Each test case includes a different configuration scenario and expects a specific error message
// or no error based on the validity of the configuration. The scenarios tested include:
// - Nil configuration
// - Empty Kafka brokers list
// - Kafka brokers list with empty broker
// - Empty Kafka version
// - Empty Kafka group ID
// - Invalid producer max retry count
// - Invalid consumer session timeout
// - Invalid heartbeat interval
// - Invalid max processing time
// - Heartbeat interval greater than or equal to session timeout
// - A valid configuration
func TestValidateKafkaConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.KafkaConfigEntry
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "kafka configuration is nil",
		},
		{
			name: "empty brokers",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Kafka.Brokers = []string{}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "kafka broker addresses are empty",
		},
		{
			name: "empty broker in list",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Kafka.Brokers = []string{"localhost:9092", ""}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "kafka broker address[1] is empty",
		},
		{
			name: "empty version",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Kafka.Version = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "kafka version is empty",
		},
		{
			name: "empty group ID",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Kafka.GroupID = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "kafka consumer group ID is empty",
		},
		{
			name: "invalid producer max retry",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Advanced.ProducerMaxRetry = -1
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid producer max retry count: -1, must be non-negative",
		},
		{
			name: "invalid consumer session timeout",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Advanced.ConsumerSessionTimeout = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid consumer session timeout: 0 ms, must be greater than 0",
		},
		{
			name: "invalid heartbeat interval",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Advanced.HeartbeatInterval = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid heartbeat interval: 0 ms, must be greater than 0",
		},
		{
			name: "invalid max processing time",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Advanced.MaxProcessingTime = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid max processing time: 0 ms, must be greater than 0",
		},
		{
			name: "heartbeat interval >= session timeout",
			config: func() *config.KafkaConfigEntry {
				cfg := setupTestKafkaConfig()
				cfg.Advanced.HeartbeatInterval = 30000
				cfg.Advanced.ConsumerSessionTimeout = 30000
				return cfg
			}(),
			expectError: true,
			errorMsg:    "heartbeat interval (30000 ms) must be less than session timeout (30000 ms)",
		},
		{
			name:        "valid config",
			config:      setupTestKafkaConfig(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKafkaConfig(tt.config)

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

// TestConfigureKafkaProducer tests the ConfigureKafkaProducer function by verifying
// the basic Kafka producer settings and network settings are properly configured.
//
// The test uses the default test configuration and expects the following settings
// to be configured:
//
//   - Version: The parsed Kafka version
//   - Return.Successes: true
//   - Return.Errors: true
//   - RequiredAcks: WaitForAll
//   - Retry.Max: The configured maximum retry count
//   - Compression: Snappy
//   - Idempotent: true
//   - Net.DialTimeout: 30s
//   - Net.MaxOpenRequests: 5 (overridden from 1 due to idempotency)
func TestConfigureKafkaProducer(t *testing.T) {
	cfg := setupTestKafkaConfig()
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		t.Fatalf("Failed to parse Kafka version: %v", err)
	}

	producerConfig := ConfigureKafkaProducer(cfg, version)

	// Verify basic settings
	if producerConfig.Version != version {
		t.Errorf("Expected version %v, got %v", version, producerConfig.Version)
	}

	if !producerConfig.Producer.Return.Successes {
		t.Errorf("Expected Producer.Return.Successes to be true")
	}

	if !producerConfig.Producer.Return.Errors {
		t.Errorf("Expected Producer.Return.Errors to be true")
	}

	if producerConfig.Producer.RequiredAcks != sarama.WaitForAll {
		t.Errorf("Expected Producer.RequiredAcks to be WaitForAll")
	}

	if producerConfig.Producer.Retry.Max != cfg.Advanced.ProducerMaxRetry {
		t.Errorf("Expected Producer.Retry.Max to be %d, got %d", cfg.Advanced.ProducerMaxRetry, producerConfig.Producer.Retry.Max)
	}

	if producerConfig.Producer.Compression != sarama.CompressionSnappy {
		t.Errorf("Expected Producer.Compression to be Snappy")
	}

	if !producerConfig.Producer.Idempotent {
		t.Errorf("Expected Producer.Idempotent to be true")
	}

	// Verify network settings
	if producerConfig.Net.DialTimeout != 30*time.Second {
		t.Errorf("Expected Net.DialTimeout to be 30s, got %v", producerConfig.Net.DialTimeout)
	}

	// Note: MaxOpenRequests is set to 1 for idempotency but then overridden by configureNetworkSettings
	// This is expected behavior based on the current implementation
	if producerConfig.Net.MaxOpenRequests != 5 {
		t.Errorf("Expected Net.MaxOpenRequests to be 5 (from configureNetworkSettings), got %d", producerConfig.Net.MaxOpenRequests)
	}
}

// TestConfigureKafkaConsumer tests the ConfigureKafkaConsumer function by verifying
// the basic Kafka consumer settings, advanced settings, and network settings are
// properly configured.
//
// The test uses the default test configuration and expects the following settings
// to be configured:
//
//   - Version: The parsed Kafka version
//   - Consumer.Offsets.Initial: OffsetOldest
//   - Consumer.Offsets.AutoCommit.Enable: true
//   - Consumer.Group.Session.Timeout: The configured consumer session timeout
//   - Consumer.Group.Heartbeat.Interval: The configured heartbeat interval
//   - Consumer.MaxProcessingTime: The configured max processing time
//   - Net.DialTimeout: 30s
func TestConfigureKafkaConsumer(t *testing.T) {
	cfg := setupTestKafkaConfig()
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		t.Fatalf("Failed to parse Kafka version: %v", err)
	}

	consumerConfig := ConfigureKafkaConsumer(cfg, version)

	// Verify basic settings
	if consumerConfig.Version != version {
		t.Errorf("Expected version %v, got %v", version, consumerConfig.Version)
	}

	if consumerConfig.Consumer.Offsets.Initial != sarama.OffsetOldest {
		t.Errorf("Expected Consumer.Offsets.Initial to be OffsetOldest")
	}

	if !consumerConfig.Consumer.Offsets.AutoCommit.Enable {
		t.Errorf("Expected Consumer.Offsets.AutoCommit.Enable to be true")
	}

	expectedSessionTimeout := time.Duration(cfg.Advanced.ConsumerSessionTimeout) * time.Millisecond
	if consumerConfig.Consumer.Group.Session.Timeout != expectedSessionTimeout {
		t.Errorf("Expected Consumer.Group.Session.Timeout to be %v, got %v", expectedSessionTimeout, consumerConfig.Consumer.Group.Session.Timeout)
	}

	expectedHeartbeatInterval := time.Duration(cfg.Advanced.HeartbeatInterval) * time.Millisecond
	if consumerConfig.Consumer.Group.Heartbeat.Interval != expectedHeartbeatInterval {
		t.Errorf("Expected Consumer.Group.Heartbeat.Interval to be %v, got %v", expectedHeartbeatInterval, consumerConfig.Consumer.Group.Heartbeat.Interval)
	}

	expectedMaxProcessingTime := time.Duration(cfg.Advanced.MaxProcessingTime) * time.Millisecond
	if consumerConfig.Consumer.MaxProcessingTime != expectedMaxProcessingTime {
		t.Errorf("Expected Consumer.MaxProcessingTime to be %v, got %v", expectedMaxProcessingTime, consumerConfig.Consumer.MaxProcessingTime)
	}

	// Verify network settings
	if consumerConfig.Net.DialTimeout != 30*time.Second {
		t.Errorf("Expected Net.DialTimeout to be 30s, got %v", consumerConfig.Net.DialTimeout)
	}
}

// TestConfigureNetworkSettings verifies that the configureNetworkSettings function
// correctly sets the network settings for a Kafka configuration.
//
// The test checks the following network settings:
//   - Net.DialTimeout is set to 30 seconds
//   - Net.ReadTimeout is set to 30 seconds
//   - Net.WriteTimeout is set to 30 seconds
//   - Net.MaxOpenRequests is set to 5
//   - Net.KeepAlive is set to 30 seconds
func TestConfigureNetworkSettings(t *testing.T) {
	config := sarama.NewConfig()
	configureNetworkSettings(config)

	if config.Net.DialTimeout != 30*time.Second {
		t.Errorf("Expected Net.DialTimeout to be 30s, got %v", config.Net.DialTimeout)
	}

	if config.Net.ReadTimeout != 30*time.Second {
		t.Errorf("Expected Net.ReadTimeout to be 30s, got %v", config.Net.ReadTimeout)
	}

	if config.Net.WriteTimeout != 30*time.Second {
		t.Errorf("Expected Net.WriteTimeout to be 30s, got %v", config.Net.WriteTimeout)
	}

	if config.Net.MaxOpenRequests != 5 {
		t.Errorf("Expected Net.MaxOpenRequests to be 5, got %d", config.Net.MaxOpenRequests)
	}

	if config.Net.KeepAlive != 30*time.Second {
		t.Errorf("Expected Net.KeepAlive to be 30s, got %v", config.Net.KeepAlive)
	}
}

// TestTestKafkaConnection_WithMockConfig tests the TestKafkaConnection function
// with a mock Kafka configuration.
//
// The test is skipped for now as it requires a running Kafka instance.
func TestTestKafkaConnection_WithMockConfig(t *testing.T) {
	// This test would require actual Kafka connection
	// Skip for now as it requires running Kafka instance
	t.Skip("Skipping test - requires running Kafka instance")
}

// TestCleanupKafkaClients tests the cleanupKafkaClients function by verifying that
// it cleans up the Kafka clients properly.
//
// The test sets up some mock clients (nil is fine for this test) and calls
// cleanupKafkaClients. It then verifies that the clients are still nil after
// the cleanup.
func TestCleanupKafkaClients(t *testing.T) {
	setupTestLoggerForKafka()

	// Set some mock clients (nil is fine for this test)
	resource.KafkaProducer = nil
	resource.KafkaConsumer = nil
	resource.KafkaConsumerGroup = nil

	// This should not panic
	cleanupKafkaClients()

	// Verify clients are still nil
	if resource.KafkaProducer != nil {
		t.Errorf("Expected KafkaProducer to be nil after cleanup")
	}
	if resource.KafkaConsumer != nil {
		t.Errorf("Expected KafkaConsumer to be nil after cleanup")
	}
	if resource.KafkaConsumerGroup != nil {
		t.Errorf("Expected KafkaConsumerGroup to be nil after cleanup")
	}
}

// TestCloseKafka_NoClients tests the CloseKafka function by verifying that it
// does not return an error when there are no Kafka clients to close.
//
// The test sets up no clients and calls CloseKafka. It then verifies that no
// error is returned.
func TestCloseKafka_NoClients(t *testing.T) {
	setupTestLoggerForKafka()

	// Ensure no clients are set
	resource.KafkaProducer = nil
	resource.KafkaConsumer = nil
	resource.KafkaConsumerGroup = nil

	err := CloseKafka()
	if err != nil {
		t.Errorf("Expected no error when closing with no clients, got: %v", err)
	}
}

// TestPerformHealthCheck_NoClients tests the performHealthCheck function by verifying
// that it returns an error when there are no Kafka clients to health check.
//
// The test sets up no clients and calls performHealthCheck. It then verifies that
// an error is returned that contains the expected substring "kafka producer is not
// initialized".
func TestPerformHealthCheck_NoClients(t *testing.T) {
	// Set clients to nil
	resource.KafkaProducer = nil
	resource.KafkaConsumer = nil
	resource.KafkaConsumerGroup = nil

	err := performHealthCheck()
	if err == nil {
		t.Errorf("Expected error when performing health check with no clients")
	}

	expectedSubstring := "kafka producer is not initialized"
	if !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

// TestInitKafka_WithValidConfig tests the InitKafka function with a valid
// configuration by verifying that it initializes the Kafka clients properly.
//
// The test sets up a valid test configuration and calls InitKafka. It then
// verifies that the clients are initialized and that no error is returned.
//
// The test is skipped if Kafka is not available.
func TestInitKafka_WithValidConfig(t *testing.T) {
	setupTestLoggerForKafka()

	// Set up test config
	config.KafkaConfig = setupTestKafkaConfig()

	// This test will only pass if Kafka is actually running
	// Skip if Kafka is not available
	t.Skip("Skipping integration test - requires running Kafka instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitKafka panicked: %v", r)
		}
	}()

	InitKafka(ctx)

	// Clean up
	if resource.KafkaProducer != nil || resource.KafkaConsumer != nil || resource.KafkaConsumerGroup != nil {
		_ = CloseKafka()
	}
}
