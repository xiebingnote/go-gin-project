package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/IBM/sarama"
)

// InitKafka initializes the Kafka client with the configuration specified in
// the./conf/kafka.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the Kafka cluster.
//
// The initialized Kafka client is stored as a singleton in the resource package
// for use throughout the application.
//
// If the configuration file decoding fails, the function panics with an error.
func InitKafka(_ context.Context) {
	if err := InitKafkaClient(); err != nil {
		panic(err.Error())
	}
}

// InitKafkaClient initializes the Kafka client with the configuration specified
// in the./conf/kafka.toml file.
//
// It reads the configuration parameters required to connect and authenticate with
// the Kafka cluster.
//
// The initialized Kafka client is stored as a singleton in the resource package for
// use throughout the application.
//
// If the configuration file decoding fails, the function panics with an error.
func InitKafkaClient() error {
	cfg := config.KafkaConfig

	// Initialize the producer
	// Create a producer configuration
	producerConfig := sarama.NewConfig()
	// Enable return of successful messages
	producerConfig.Producer.Return.Successes = true
	// Enable return of errors
	producerConfig.Producer.Return.Errors = true
	// More reliable acknowledgment mechanism
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	// Increase the connection timeout
	producerConfig.Net.DialTimeout = 30 * time.Second
	// Increase the maximum number of retries
	producerConfig.Producer.Retry.Max = config.KafkaConfig.Advanced.ProducerMaxRetry

	// Parse the Kafka version string
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		return fmt.Errorf("invalid kafka version: %w", err)
	}
	producerConfig.Version = version

	// Create a synchronous producer
	resource.KafkaProducer, err = sarama.NewSyncProducer(cfg.Kafka.Brokers, producerConfig)
	if err != nil {
		return fmt.Errorf("kafka producer init failed: %w", err)
	}

	// Initialize the consumer
	// Create a consumer configuration
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = version
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		// Use range partitioning strategy
		sarama.NewBalanceStrategyRange(),
	}

	// Initialize the consumer
	resource.KafkaConsumer, err = sarama.NewConsumer(cfg.Kafka.Brokers, consumerConfig)
	if err != nil {
		return fmt.Errorf("kafka consumer init failed: %w", err)
	}

	// Initialize the consumer group
	resource.KafkaConsumerGroup, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, consumerConfig)
	if err != nil {
		return fmt.Errorf("kafka consumer group init failed: %w", err)
	}

	// Start a background health check
	go kafkaHealthCheck(context.Background())

	//resource.LoggerService.Info(fmt.Sprintf("Kafka initialized | Brokers: %v | Version: %s", cfg.Kafka.Brokers, cfg.Kafka.Version))
	return nil
}

// CloseKafka closes the Kafka producer and consumer, and returns an error if either
// close operation fails.
//
// The function returns a single error that combines all the errors that occurred
// during the shutdown process. If all the close operations succeed, the function
// returns nil.
func CloseKafka() error {
	var errs []error

	// Close the producer
	if resource.KafkaProducer != nil {
		if err := resource.KafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("producer close failed: %w", err))
		}
	}

	// Close the consumer
	if resource.KafkaConsumer != nil {
		if err := resource.KafkaConsumer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("consumer close failed: %w", err))
		}
	}

	// Close the consumer group
	if resource.KafkaConsumerGroup != nil {
		if err := resource.KafkaConsumerGroup.Close(); err != nil {
			errs = append(errs, fmt.Errorf("consumer group close failed: %w", err))
		}
	}

	// If any errors occurred during the shutdown process, return the combined error
	if len(errs) > 0 {
		return fmt.Errorf("kafka shutdown errors: %v", errs)
	}
	return nil
}

// kafkaHealthCheck performs a periodic health check on the Kafka producer and consumer.
//
// This function runs in a continuous loop, sending a "ping" message to the Kafka producer
// and logging the status of both the producer and consumer. It stops when the context is done.
func kafkaHealthCheck(ctx context.Context) {
	// Create a ticker that triggers every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check the health of the Kafka producer
			if resource.KafkaProducer != nil {
				// Send a "ping" message to a health check topic
				if _, _, err := resource.KafkaProducer.SendMessage(&sarama.ProducerMessage{
					Topic: "health_check_topic",
					Value: sarama.StringEncoder("ping"),
				}); err != nil {
					resource.LoggerService.Error(fmt.Sprintf("Kafka producer health check failed: %v", err))
					return
				}
			}

		case <-ctx.Done():
			// Exit the loop when context is canceled or times out
			return
		}
	}
}
