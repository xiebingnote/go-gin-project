package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/IBM/sarama"
)

var (
	// healthCheckCtx is the context used for health checks.
	healthCheckCtx    context.Context
	healthCheckCancel context.CancelFunc
	healthCheckOnce   sync.Once
)

// InitKafka initializes the Kafka client connection.
//
// This function attempts to establish a new Kafka client connection
// using the provided context. If the connection cannot be established,
// it logs an error message and panics with the error.
//
// The context parameter is used to control the lifecycle of the
// Kafka client connection. If the context is canceled, the connection
// will also be canceled.
//
// Parameters:
//   - ctx: context.Context used for managing request-scoped values
//     and cancellation signals.
//
// Panics:
//   - If the Kafka client connection cannot be initialized.
func InitKafka(ctx context.Context) {
	if err := InitKafkaClient(ctx); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize kafka: %v", err))
		panic(err.Error())
	}
}

// InitKafkaClient initializes a new Kafka client connection pool.
//
// This function performs the following steps:
// 1. Validates the Kafka configuration
// 2. Configures the producer and consumer
// 3. Creates a Kafka client
// 4. Tests the connection
// 5. Stores the initialized client in the global resource
// 6. Starts the background health check
//
// If the configuration is invalid, the connection cannot be established,
// or any of the steps fail, it returns an error.
func InitKafkaClient(ctx context.Context) error {
	cfg := config.KafkaConfig

	// Validate the Kafka configuration
	if err := ValidateKafkaConfig(cfg); err != nil {
		return fmt.Errorf("invalid kafka configuration: %w", err)
	}

	// Parse the Kafka version
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		return fmt.Errorf("invalid kafka version: %w", err)
	}

	// Initialize the producer
	producerConfig := ConfigureKafkaProducer(cfg, version)
	resource.KafkaProducer, err = sarama.NewSyncProducer(cfg.Kafka.Brokers, producerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize kafka producer: %w", err)
	}

	// Initialize the consumer
	consumerConfig := ConfigureKafkaConsumer(cfg, version)
	resource.KafkaConsumer, err = sarama.NewConsumer(cfg.Kafka.Brokers, consumerConfig)
	if err != nil {
		// If the consumer initialization fails, clean up the producer
		if resource.KafkaProducer != nil {
			if closeErr := resource.KafkaProducer.Close(); closeErr != nil {
				resource.LoggerService.Error(fmt.Sprintf("failed to close kafka producer during cleanup: %v", closeErr))
			}
			resource.KafkaProducer = nil
		}
		return fmt.Errorf("failed to initialize kafka consumer: %w", err)
	}

	// Initialize the consumer group
	resource.KafkaConsumerGroup, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, consumerConfig)
	if err != nil {
		// If the consumer group initialization fails, clean up all the clients
		cleanupKafkaClients()
		return fmt.Errorf("failed to initialize kafka consumer group: %w", err)
	}

	// Test the connection (using a lighter-weight approach)
	if err := TestKafkaConnection(cfg); err != nil {
		cleanupKafkaClients()
		return fmt.Errorf("kafka connection test failed: %w", err)
	}

	// Start the background health check
	startKafkaHealthCheck(ctx)

	resource.LoggerService.Info(fmt.Sprintf("successfully connected to kafka | brokers: %v | version: %s",
		cfg.Kafka.Brokers, cfg.Kafka.Version))
	return nil
}

// cleanupKafkaClients clears the Kafka clients that have been created.
//
// This function is used to clean up the Kafka clients when the initialization
// of the Kafka clients fails. It closes the clients and sets them to nil.
// All close errors are logged but do not prevent the cleanup from continuing.
func cleanupKafkaClients() {
	// Close the producer if it has been initialized
	if resource.KafkaProducer != nil {
		if err := resource.KafkaProducer.Close(); err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close kafka producer during cleanup: %v", err))
		}
		resource.KafkaProducer = nil
	}

	// Close the consumer if it has been initialized
	if resource.KafkaConsumer != nil {
		if err := resource.KafkaConsumer.Close(); err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close kafka consumer during cleanup: %v", err))
		}
		resource.KafkaConsumer = nil
	}

	// Close the consumer group if it has been initialized
	if resource.KafkaConsumerGroup != nil {
		if err := resource.KafkaConsumerGroup.Close(); err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close kafka consumer group during cleanup: %v", err))
		}
		resource.KafkaConsumerGroup = nil
	}
}

// ValidateKafkaConfig validates the Kafka configuration.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Brokers (Kafka broker addresses)
//     - Version (Kafka version)
//     - GroupID (Kafka consumer group ID)
//  3. Advanced settings are valid:
//     - ProducerMaxRetry (maximum retry count for the producer)
//     - ConsumerSessionTimeout (session timeout for the consumer)
//     - HeartbeatInterval (heartbeat interval for the consumer)
//     - MaxProcessingTime (maximum processing time for the consumer)
func ValidateKafkaConfig(cfg *config.KafkaConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("kafka configuration is nil")
	}

	// Check required connection parameters
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka broker addresses are empty")
	}

	// Validate each broker address
	for i, broker := range cfg.Kafka.Brokers {
		if broker == "" {
			return fmt.Errorf("kafka broker address[%d] is empty", i)
		}
	}

	if cfg.Kafka.Version == "" {
		return fmt.Errorf("kafka version is empty")
	}
	if cfg.Kafka.GroupID == "" {
		return fmt.Errorf("kafka consumer group ID is empty")
	}

	// Check advanced settings with detailed error messages
	if cfg.Advanced.ProducerMaxRetry < 0 {
		return fmt.Errorf("invalid producer max retry count: %d, must be non-negative", cfg.Advanced.ProducerMaxRetry)
	}
	if cfg.Advanced.ConsumerSessionTimeout <= 0 {
		return fmt.Errorf("invalid consumer session timeout: %d ms, must be greater than 0", cfg.Advanced.ConsumerSessionTimeout)
	}
	if cfg.Advanced.HeartbeatInterval <= 0 {
		return fmt.Errorf("invalid heartbeat interval: %d ms, must be greater than 0", cfg.Advanced.HeartbeatInterval)
	}
	if cfg.Advanced.MaxProcessingTime <= 0 {
		return fmt.Errorf("invalid max processing time: %d ms, must be greater than 0", cfg.Advanced.MaxProcessingTime)
	}

	// Check logical consistency
	if cfg.Advanced.HeartbeatInterval >= cfg.Advanced.ConsumerSessionTimeout {
		return fmt.Errorf("heartbeat interval (%d ms) must be less than session timeout (%d ms)",
			cfg.Advanced.HeartbeatInterval, cfg.Advanced.ConsumerSessionTimeout)
	}

	return nil
}

// configureNetworkSettings sets the network settings for the Kafka configuration.
//
// The function sets the following network settings:
//
//  1. Dial timeout: 30 seconds
//  2. Read timeout: 30 seconds
//  3. Write timeout: 30 seconds
//  4. Maximum open requests: 5
//  5. Keep alive: 30 seconds
//
// Parameters:
//   - config: the Kafka configuration to set the network settings for
func configureNetworkSettings(config *sarama.Config) {
	config.Net.DialTimeout = 30 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second
	config.Net.MaxOpenRequests = 5
	config.Net.KeepAlive = 30 * time.Second
}

// ConfigureKafkaProducer configures Kafka producer options
//
// Parameters:
//   - cfg: A pointer to the Kafka configuration containing connection settings
//   - version: The parsed Kafka version
//
// Returns:
//   - A configured *sarama.Config producer instance
func ConfigureKafkaProducer(cfg *config.KafkaConfigEntry, version sarama.KafkaVersion) *sarama.Config {
	configSarama := sarama.NewConfig()
	configSarama.Version = version

	// Enable returning successes and errors for message delivery
	configSarama.Producer.Return.Successes = true
	configSarama.Producer.Return.Errors = true

	// Set required acks to wait for all replicas to acknowledge
	configSarama.Producer.RequiredAcks = sarama.WaitForAll

	// Set the maximum number of retries for failed messages
	configSarama.Producer.Retry.Max = cfg.Advanced.ProducerMaxRetry

	// Use Snappy compression for messages
	configSarama.Producer.Compression = sarama.CompressionSnappy

	// Set flush frequency and limits
	configSarama.Producer.Flush.Frequency = 500 * time.Millisecond
	configSarama.Producer.Flush.MaxMessages = 100
	configSarama.Producer.Flush.Bytes = 1024 * 1024 // 1MB

	// Apply common network settings first
	configureNetworkSettings(configSarama)

	// Enable idempotency to avoid duplicate messages
	configSarama.Producer.Idempotent = true

	// Set max open requests to 1 for idempotency (must be after configureNetworkSettings)
	configSarama.Net.MaxOpenRequests = 1

	return configSarama
}

// ConfigureKafkaConsumer configures Kafka consumer options
//
// Parameters:
//   - cfg: a pointer to the Kafka configuration containing connection settings
//   - version: the parsed Kafka version
//
// Returns:
//   - a configured *sarama.Config consumer instance
func ConfigureKafkaConsumer(cfg *config.KafkaConfigEntry, version sarama.KafkaVersion) *sarama.Config {
	configSarama := sarama.NewConfig()
	configSarama.Version = version

	// Consumer settings
	//
	//  1. Set the initial offset to the oldest available message
	//  2. Enable auto-commit with an interval of 1 second
	//  3. Set the group rebalance strategy to round-robin
	//  4. Set the session timeout to the value in the configuration file
	//  5. Set the heartbeat interval to the value in the configuration file
	//  6. Set the maximum processing time to the value in the configuration file
	//  7. Set the maximum wait time to 250 milliseconds
	//  8. Set the minimum and maximum fetch sizes to 1MB and 10MB respectively
	configSarama.Consumer.Offsets.Initial = sarama.OffsetOldest
	configSarama.Consumer.Offsets.AutoCommit.Enable = true
	configSarama.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	configSarama.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
		sarama.NewBalanceStrategyRoundRobin(), // add round-robin strategy for better load balancing
	}

	configSarama.Consumer.Group.Session.Timeout = time.Duration(cfg.Advanced.ConsumerSessionTimeout) * time.Millisecond
	configSarama.Consumer.Group.Heartbeat.Interval = time.Duration(cfg.Advanced.HeartbeatInterval) * time.Millisecond
	configSarama.Consumer.MaxProcessingTime = time.Duration(cfg.Advanced.MaxProcessingTime) * time.Millisecond

	configSarama.Consumer.MaxWaitTime = 250 * time.Millisecond
	configSarama.Consumer.Fetch.Min = 1
	configSarama.Consumer.Fetch.Default = 1024 * 1024  // 1MB
	configSarama.Consumer.Fetch.Max = 1024 * 1024 * 10 // 10MB

	// Apply common network settings
	configureNetworkSettings(configSarama)

	return configSarama
}

// TestKafkaConnection tests the Kafka connection by retrieving metadata.
//
// This function tests the connection by getting the cluster metadata, which is
// a lightweight and non-intrusive way to test the connection without sending
// actual messages.
//
// Parameters:
//   - cfg: a pointer to the Kafka configuration containing connection settings
//
// Returns:
//   - an error if the connection test fails
//   - nil if the connection test succeeds
func TestKafkaConnection(cfg *config.KafkaConfigEntry) error {
	// Create a temporary client to test the connection
	configKafka := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		return fmt.Errorf("failed to parse kafka version: %w", err)
	}
	configKafka.Version = version

	// Configure the network settings
	configureNetworkSettings(configKafka)

	// Create the temporary client
	client, err := sarama.NewClient(cfg.Kafka.Brokers, configKafka)
	if err != nil {
		return fmt.Errorf("failed to create test client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close test client: %v", closeErr))
		}
	}()

	// Test the connection by retrieving the cluster metadata
	err = client.RefreshMetadata()
	if err != nil {
		return fmt.Errorf("failed to retrieve kafka metadata: %w", err)
	}

	return nil
}

// CloseKafka closes all Kafka connections.
//
// This function attempts to close all Kafka connections, including the producer,
// consumer, and consumer group. If any of the close operations fail, it collects
// the errors and returns a combined error message.
//
// Returns:
//   - an error if any of the close operations fail
//   - nil if all the close operations succeed
func CloseKafka() error {
	var errs []error

	// Stop the health check goroutine
	stopKafkaHealthCheck()

	// Close the producer
	if resource.KafkaProducer != nil {
		if err := resource.KafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close kafka producer: %w", err))
		}
		resource.KafkaProducer = nil
	}

	// Close the consumer
	if resource.KafkaConsumer != nil {
		if err := resource.KafkaConsumer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close kafka consumer: %w", err))
		}
		resource.KafkaConsumer = nil
	}

	// Close the consumer group
	if resource.KafkaConsumerGroup != nil {
		if err := resource.KafkaConsumerGroup.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close kafka consumer group: %w", err))
		}
		resource.KafkaConsumerGroup = nil
	}

	// If any of the close operations fail, return a combined error
	if len(errs) > 0 {
		return fmt.Errorf("failed to close all kafka connections: %v", errs)
	}

	// Log a success message to indicate that all connections have been closed
	resource.LoggerService.Info("successfully closed all kafka connections")
	return nil
}

// startKafkaHealthCheck starts the Kafka health check goroutine.
//
// This function starts a goroutine that periodically checks the health of the Kafka
// connection. The goroutine will stop when the context is canceled.
// The healthCheckOnce variable ensures that the health check goroutine is only
// started once.
func startKafkaHealthCheck(ctx context.Context) {
	healthCheckOnce.Do(func() {
		healthCheckCtx, healthCheckCancel = context.WithCancel(ctx)
		go kafkaHealthCheck(healthCheckCtx)
	})
}

// stopKafkaHealthCheck stops the Kafka health check goroutine.
//
// This function stops the Kafka health check goroutine. It is safe to call
// this function multiple times.
//
// This function is used to stop the health check goroutine when the application
// is being shut down.
func stopKafkaHealthCheck() {
	if healthCheckCancel != nil {
		healthCheckCancel()
		healthCheckCancel = nil
	}
}

// kafkaHealthCheck performs periodic health checks for Kafka.
//
// This function runs in a continuous loop, monitoring the health status by checking client states.
// It avoids sending actual messages, using lighter checks instead.
func kafkaHealthCheck(ctx context.Context) {
	// Create a ticker that triggers every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop() // Ensure the ticker is stopped when the function exits

	// Log that the Kafka health check has started
	resource.LoggerService.Info("kafka health check started")

	// Listen for ticker signals to perform health checks periodically
	for {
		select {
		case <-ticker.C:
			// Perform the health check
			if err := performHealthCheck(); err != nil {
				// Log an error if the health check fails
				resource.LoggerService.Error(fmt.Sprintf("kafka health check failed: %v", err))
				// Reconnection logic or alerting can be added here
			} else {
				// Log a debug message if the health check passes
				resource.LoggerService.Debug("kafka health check passed")
			}
		case <-ctx.Done():
			// If the context is canceled, stop the health check
			resource.LoggerService.Info("kafka health check stopped")
			return
		}
	}
}

// performHealthCheck performs the actual health check.
//
// This function checks the health of the Kafka connection by checking the states
// of the producer, consumer, and consumer group.
func performHealthCheck() error {
	// Check the producer status
	if resource.KafkaProducer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Check the consumer status
	if resource.KafkaConsumer == nil {
		return fmt.Errorf("kafka consumer is not initialized")
	}

	// Check the consumer group status
	if resource.KafkaConsumerGroup == nil {
		return fmt.Errorf("kafka consumer group is not initialized")
	}

	// More checks can be added here, such as checking the connection status
	return nil
}
