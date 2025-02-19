package service

import (
	"context"
	"fmt"

	libconfig "go-gin-project/library/config"
	"go-gin-project/library/resource"

	"github.com/nsqio/go-nsq"
)

// InitNSQ initializes the NSQ client.
//
// This function calls InitNSQClient to set up the NSQ producers and consumers
// using the configuration provided.
//
// If the initialization fails, it panics with an error message.
func InitNSQ(_ context.Context) {
	// Attempt to initialize the NSQ client
	err := InitNSQClient()
	if err != nil {
		// Panic if there is an error initializing the NSQ client
		panic(err.Error())
	}
}

// InitNSQClient initializes the NSQ client using the configuration provided
// in the global libconfig.NsqConfig.
//
// It first checks if the NSQLookupdAddress configuration is empty.
//
// If it is, it logs an error and returns an error.
//
// Then, it calls InitProducers and InitConsumer to set up the NSQ producers
// and consumers using the configuration provided.
//
// If either of these functions fails, it logs an error and returns the error.
func InitNSQClient() error {
	if len(libconfig.NsqConfig.NSQ.LookupdAddress) == 0 {
		resource.LoggerService.Error("NSQLookupdAddress are nil")
		return fmt.Errorf("NSQLookupdAddress are nil")
	}

	// Initialize the NSQ producers.
	if err := InitProducers(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize producer, err: %v", err))
		return fmt.Errorf("failed to initialize producer: %v", err)
	}

	// Initialize the NSQ consumer.
	if err := InitConsumer(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize consumer, err: %v", err))
		return fmt.Errorf("failed to initialize consumer: %v", err)
	}

	return nil
}

// InitProducers initializes the NSQ producers using the configuration provided
// in the global libconfig.NsqConfig.
//
// It creates a new NSQ config with the specified dial timeout and maximum attempts.
//
// Then, for each address in the NSQLookupdAddress configuration, it
// creates a new NSQ producer using the config.
//
// If any of the producers fail to be created, it logs an error and returns the error.
//
// After creating the producers, it tests the connection by sending a ping to
// each producer.
//
// If any of the pings fail, it logs an error and returns the error.
func InitProducers() error {
	config := nsq.NewConfig()
	config.DialTimeout = libconfig.NsqConfig.NSQ.Producer.DialTimeout
	config.MaxAttempts = uint16(libconfig.NsqConfig.NSQ.Producer.MaxAttempts)

	for _, addr := range libconfig.NsqConfig.NSQ.LookupdAddress {
		producer, err := nsq.NewProducer(addr, config)
		if err != nil {
			// Log an error if the producer fails to be created.
			resource.LoggerService.Error(fmt.Sprintf(
				"failed to create producer for address %s, err: %v",
				addr,
				err,
			))
			return fmt.Errorf(
				"failed to create producer for address %s: %w",
				addr,
				err,
			)
		}

		// Test the connection by sending a ping to the producer.
		if err := producer.Ping(); err != nil {
			// Log an error if the ping fails.
			resource.LoggerService.Error(fmt.Sprintf(
				"producer ping failed for address %s, err: %v",
				addr,
				err,
			))
			return fmt.Errorf(
				"producer ping failed for address %s: %w",
				addr,
				err,
			)
		}

		// Add the producer to the list of producers.
		resource.NsqProducer = append(resource.NsqProducer, producer)
	}

	return nil
}

// InitConsumer initializes an NSQ consumer with the specified configuration.
//
// It sets up the consumer configuration with the maximum number of inflight
// messages, maximum attempts, default requeue delay, and heartbeat interval.
//
// A new NSQ consumer is then created using the topic, channel, and configuration.
//
// If the consumer creation fails, the function returns an error.
//
// Otherwise, it assigns the consumer to the global NsqConsumer resource.
func InitConsumer() error {
	// Create a new NSQ configuration
	config := nsq.NewConfig()
	// Set the maximum number of inflight messages.
	config.MaxInFlight = libconfig.NsqConfig.NSQ.Consumer.MaxInFlight
	// Set the maximum number of times the consumer will retry a message.
	config.MaxAttempts = uint16(libconfig.NsqConfig.NSQ.Consumer.MaxAttempts)
	// Set the default requeue delay.
	config.DefaultRequeueDelay = libconfig.NsqConfig.NSQ.Consumer.RequeueDelay
	// Set the heartbeat interval.
	config.HeartbeatInterval = libconfig.NsqConfig.NSQ.Consumer.HeartbeatInterval

	// Create a new NSQ consumer with the specified topic, channel, and configuration
	consumer, err := nsq.NewConsumer(
		libconfig.NsqConfig.NSQ.Consumer.Topic,
		libconfig.NsqConfig.NSQ.Consumer.Channel,
		config,
	)
	if err != nil {
		// Return the error if consumer creation fails
		return err
	}

	// Assign the consumer to the global NsqConsumer resource
	resource.NsqConsumer = consumer
	return nil
}

// CloseNsq closes all the NSQ connections safely.
//
// It stops all the producers and the consumer by calling their respective Stop
// methods.
func CloseNsq() error {
	// Stop all the producers
	for _, producer := range resource.NsqProducer {
		producer.Stop()
	}

	// Log that all the producers have been stopped
	resource.LoggerService.Info("All NSQ producer stopped")

	// Stop the consumer
	if resource.NsqConsumer != nil {
		resource.NsqConsumer.Stop()
		// Log that the consumer has been stopped
		resource.LoggerService.Info("NSQ consumer stopped")
	}

	return nil
}
