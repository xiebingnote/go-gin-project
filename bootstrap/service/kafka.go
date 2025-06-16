package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/IBM/sarama"
)

// InitKafka initializes the Kafka client with the configuration
// specified in the ./conf/kafka.toml file.
func InitKafka(_ context.Context) {
	if err := InitKafkaClient(); err != nil {
		// Log an error message if the Kafka connection cannot be established
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize Kafka: %v", err))
		panic(err.Error())
	}
}

// InitKafkaClient initializes a new Kafka client with connection pool.
//
// The function performs the following steps:
// 1. Validates the Kafka configuration
// 2. Configures the producer and consumer
// 3. Creates the Kafka clients
// 4. Tests the connection
// 5. Stores the initialized clients in the global resource
//
// Returns an error if the configuration is invalid, the connection cannot be established,
// or any of the steps fail.
func InitKafkaClient() error {
	cfg := config.KafkaConfig

	// Validate the Kafka configuration
	if err := ValidateKafkaConfig(cfg); err != nil {
		return fmt.Errorf("invalid Kafka configuration: %w", err)
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
		return fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}

	// Initialize the consumer
	consumerConfig := ConfigureKafkaConsumer(cfg, version)
	resource.KafkaConsumer, err = sarama.NewConsumer(cfg.Kafka.Brokers, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}

	// Initialize the consumer group
	resource.KafkaConsumerGroup, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Kafka consumer group: %w", err)
	}

	// Test the connection
	if err := TestKafkaConnection(resource.KafkaProducer); err != nil {
		return fmt.Errorf("failed to test Kafka connection: %w", err)
	}

	// Start a background health check
	go kafkaHealthCheck(context.Background())

	resource.LoggerService.Info(fmt.Sprintf("Successfully connected to Kafka | Brokers: %v | Version: %s",
		cfg.Kafka.Brokers, cfg.Kafka.Version))
	return nil
}

// ValidateKafkaConfig validates the Kafka configuration.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Brokers
//     - Version
//     - GroupID
//  3. Advanced settings are valid:
//     - ProducerMaxRetry
func ValidateKafkaConfig(cfg *config.KafkaConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("kafka configuration is nil")
	}

	// Check required connection parameters
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers are empty")
	}
	if cfg.Kafka.Version == "" {
		return fmt.Errorf("kafka version is empty")
	}
	if cfg.Kafka.GroupID == "" {
		return fmt.Errorf("kafka group ID is empty")
	}

	// Check advanced settings
	if cfg.Advanced.ProducerMaxRetry < 0 {
		return fmt.Errorf("invalid producer max retry")
	}

	return nil
}

// ConfigureKafkaProducer configures the Kafka producer options.
//
// Parameters:
//   - cfg: A pointer to the Kafka configuration containing the connection settings.
//   - version: The parsed Kafka version.
//
// Returns:
//   - A configured *sarama.Config instance for the producer.
func ConfigureKafkaProducer(cfg *config.KafkaConfigEntry, version sarama.KafkaVersion) *sarama.Config {
	configSarama := sarama.NewConfig()
	configSarama.Version = version

	// Producer settings
	configSarama.Producer.Return.Successes = true
	configSarama.Producer.Return.Errors = true
	configSarama.Producer.RequiredAcks = sarama.WaitForAll
	configSarama.Producer.Retry.Max = cfg.Advanced.ProducerMaxRetry
	configSarama.Producer.Compression = sarama.CompressionSnappy
	configSarama.Producer.Flush.Frequency = 500 * time.Millisecond
	configSarama.Producer.Flush.MaxMessages = 100
	configSarama.Producer.Flush.Bytes = 1024 * 1024 // 1MB

	// Network settings
	configSarama.Net.DialTimeout = 30 * time.Second
	configSarama.Net.ReadTimeout = 30 * time.Second
	configSarama.Net.WriteTimeout = 30 * time.Second
	configSarama.Net.MaxOpenRequests = 5
	configSarama.Net.KeepAlive = 30 * time.Second

	return configSarama
}

// ConfigureKafkaConsumer configures the Kafka consumer options.
//
// Parameters:
//   - cfg: A pointer to the Kafka configuration containing the connection settings.
//   - version: The parsed Kafka version.
//
// Returns:
//   - A configured *sarama.Config instance for the consumer.
func ConfigureKafkaConsumer(_ *config.KafkaConfigEntry, version sarama.KafkaVersion) *sarama.Config {
	configSarama := sarama.NewConfig()
	configSarama.Version = version

	// Consumer settings
	configSarama.Consumer.Offsets.Initial = sarama.OffsetOldest
	configSarama.Consumer.Offsets.AutoCommit.Enable = true
	configSarama.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	configSarama.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	configSarama.Consumer.Group.Session.Timeout = 20 * time.Second
	configSarama.Consumer.Group.Heartbeat.Interval = 6 * time.Second
	configSarama.Consumer.MaxWaitTime = 250 * time.Millisecond
	configSarama.Consumer.MaxProcessingTime = 100 * time.Millisecond
	configSarama.Consumer.Fetch.Min = 1
	configSarama.Consumer.Fetch.Default = 1024 * 1024  // 1MB
	configSarama.Consumer.Fetch.Max = 1024 * 1024 * 10 // 10MB

	// Network settings
	configSarama.Net.DialTimeout = 30 * time.Second
	configSarama.Net.ReadTimeout = 30 * time.Second
	configSarama.Net.WriteTimeout = 30 * time.Second
	configSarama.Net.MaxOpenRequests = 5
	configSarama.Net.KeepAlive = 30 * time.Second

	return configSarama
}

// TestKafkaConnection tests the Kafka connection.
//
// The function takes a Kafka producer instance as a parameter and tests
// the connection by sending a test message. If the send fails,
// the function returns an error.
//
// Parameters:
//   - producer: A Kafka producer instance to test.
//
// Returns:
//   - An error if the test message send fails
//   - nil if the test message send succeeds
func TestKafkaConnection(producer sarama.SyncProducer) error {
	// Create a test message
	msg := &sarama.ProducerMessage{
		Topic: "test_topic",
		Value: sarama.StringEncoder("test_message"),
	}

	// Send the test message
	_, _, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}

	return nil
}

// CloseKafka closes the Kafka connections.
//
// This function attempts to close all Kafka connections (producer, consumer, and consumer group).
// If any of the close operations fail, it collects the errors and returns them as a combined error.
//
// Returns:
//   - An error if any of the close operations fail
//   - nil if all close operations succeed
func CloseKafka() error {
	var errs []error

	// Close the producer
	if resource.KafkaProducer != nil {
		if err := resource.KafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Kafka producer: %w", err))
		}
		resource.KafkaProducer = nil
	}

	// Close the consumer
	if resource.KafkaConsumer != nil {
		if err := resource.KafkaConsumer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Kafka consumer: %w", err))
		}
		resource.KafkaConsumer = nil
	}

	// Close the consumer group
	if resource.KafkaConsumerGroup != nil {
		if err := resource.KafkaConsumerGroup.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Kafka consumer group: %w", err))
		}
		resource.KafkaConsumerGroup = nil
	}

	// If any errors occurred during the shutdown process, return the combined error
	if len(errs) > 0 {
		return fmt.Errorf("Kafka shutdown errors: %v", errs)
	}

	resource.LoggerService.Info("Successfully closed all Kafka connections")
	return nil
}

// kafkaHealthCheck performs a periodic health check on the Kafka producer.
//
// This function runs in a continuous loop, sending a "ping" message to the Kafka producer
// and logging the status. It stops when the context is done.
func kafkaHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if resource.KafkaProducer != nil {
				if _, _, err := resource.KafkaProducer.SendMessage(&sarama.ProducerMessage{
					Topic: "health_check_topic",
					Value: sarama.StringEncoder("ping"),
				}); err != nil {
					resource.LoggerService.Error(fmt.Sprintf("Kafka producer health check failed: %v", err))
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
