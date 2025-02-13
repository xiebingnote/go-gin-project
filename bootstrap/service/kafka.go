package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-gin-project/library/config"
	"go-gin-project/library/resource"

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
	err := InitKafkaClient()
	if err != nil {
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
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Return.Errors = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll // More reliable acknowledgment mechanism
	producerConfig.Net.DialTimeout = 30 * time.Second        // Increase the connection timeout

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
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = version
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(), // Use range partitioning strategy
	}

	// Create a consumer group
	resource.KafkaConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, consumerConfig)
	if err != nil {
		return fmt.Errorf("kafka consumer init failed: %w", err)
	}

	// Start a background health check
	go kafkaHealthCheck(context.Background())

	log.Printf("Kafka initialized | Brokers: %v | Version: %s", cfg.Kafka.Brokers, cfg.Kafka.Version)
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
					log.Printf("Kafka producer health check failed: %v", err)
				} else {
					log.Println("Kafka producer is healthy")
				}
			}

			// Check the health of the Kafka consumer
			if resource.KafkaConsumer != nil {
				log.Println("Kafka consumer is running")
			}

		case <-ctx.Done():
			// Exit the loop when context is canceled or times out
			return
		}
	}
}

type KafkaConsumerHandler struct {
	Ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
// It closes the Ready channel to signal that the consumer is ready.
func (h *KafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	// Close the Ready channel to indicate readiness
	close(h.Ready)
	return nil
}

// Cleanup is called once all ConsumeClaim goroutines have exited.
// It is a last chance to clean up any resources, but it is not
// a guarantee that it will be called in all cases (e.g., if the
// process is killed).
func (h *KafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim is called once for each consumer claim being consumed.
// A claim is a unique partition of a topic that the consumer is responsible
// for consuming. The claim is closed when the consumer is done consuming.
//
// This function will be called for each message in the partition until the
// message queue is empty, at which point the claim will be closed.
func (h *KafkaConsumerHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	// Iterate over the messages in the partition
	for message := range claim.Messages() {
		// Log the message
		log.Printf("Message received: Topic=%s Partition=%d Offset=%d Key=%s Value=%s",
			message.Topic, message.Partition, message.Offset,
			string(message.Key), string(message.Value))

		// Mark the message as processed
		session.MarkMessage(message, "")
	}
	return nil
}

// StartKafkaConsumer starts a Kafka consumer that consumes messages from the specified topics.
//
// It will block until the consumer is ready to consume messages.
func StartKafkaConsumer(topics []string) {
	handler := &KafkaConsumerHandler{
		Ready: make(chan bool),
	}

	// Start the consumer in a separate goroutine
	go func() {
		for {
			// Consume messages from the specified topics
			if err := resource.KafkaConsumer.Consume(context.Background(), topics, handler); err != nil {
				log.Printf("Consumer error: %v", err)
			}
		}
	}()

	// Wait until the consumer is ready
	<-handler.Ready
	log.Println("Kafka consumer up and running...")
}
