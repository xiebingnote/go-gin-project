package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/nsqio/go-nsq"
)

// InitNSQ initializes the NSQ client.
//
// This function calls InitNSQClient to set up the NSQ producers and consumers
// using the configuration provided.
//
// If the initialization fails, it panics with an error message.
func InitNSQ(ctx context.Context) {
	// Attempt to initialize the NSQ client with context
	if err := InitNSQClient(ctx); err != nil {
		// Log the error before panicking
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize NSQ: %v", err))
		panic(fmt.Sprintf("NSQ initialization failed: %v", err))
	}
}

// InitNSQClient initializes the NSQ client using the configuration provided
// in the global config.NsqConfig.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates the NSQ configuration
// 2. Initializes NSQ producers with connection testing
// 3. Initializes NSQ consumer with configuration validation
// 4. Performs health checks on all connections
func InitNSQClient(ctx context.Context) error {
	// Validate configuration
	if err := validateNSQConfig(); err != nil {
		return fmt.Errorf("NSQ configuration validation failed: %w", err)
	}

	resource.LoggerService.Info("Starting NSQ client initialization")

	// Initialize the NSQ producers with context
	if err := InitProducers(ctx); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize NSQ producers: %v", err))
		// Clean up any partially initialized resources
		cleanupNSQResources()
		return fmt.Errorf("failed to initialize NSQ producers: %w", err)
	}

	// Initialize the NSQ consumer with context
	if err := InitConsumer(ctx); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize NSQ consumer: %v", err))
		// Clean up any partially initialized resources
		cleanupNSQResources()
		return fmt.Errorf("failed to initialize NSQ consumer: %w", err)
	}

	resource.LoggerService.Info(fmt.Sprintf("✅ successfully initialized NSQ client with %d producers and 1 consumer",
		len(resource.NsqProducer)))

	return nil
}

// validateNSQConfig validates the NSQ configuration parameters.
//
// Returns:
//   - error: An error if any required configuration is missing or invalid, nil otherwise
func validateNSQConfig() error {
	if config.NsqConfig == nil {
		return fmt.Errorf("NSQ configuration is not initialized")
	}

	cfg := &config.NsqConfig.NSQ

	// Validate NSQ addresses
	if len(cfg.Address) == 0 {
		return fmt.Errorf("NSQ addresses are not configured")
	}

	// Validate NSQLookupd addresses
	if len(cfg.LookupdAddress) == 0 {
		return fmt.Errorf("NSQLookupd addresses are not configured")
	}

	// Validate producer configuration
	if cfg.Producer.DialTimeout <= 0 {
		return fmt.Errorf("producer dial timeout must be greater than 0")
	}
	if cfg.Producer.MaxAttempts <= 0 {
		return fmt.Errorf("producer max attempts must be greater than 0")
	}

	// Validate consumer configuration
	if cfg.Consumer.Topic == "" {
		return fmt.Errorf("consumer topic is not configured")
	}
	if cfg.Consumer.Channel == "" {
		return fmt.Errorf("consumer channel is not configured")
	}
	if cfg.Consumer.MaxInFlight <= 0 {
		return fmt.Errorf("consumer max in flight must be greater than 0")
	}
	if cfg.Consumer.MaxAttempts <= 0 {
		return fmt.Errorf("consumer max attempts must be greater than 0")
	}

	return nil
}

// cleanupNSQResources cleans up any partially initialized NSQ resources.
func cleanupNSQResources() {
	// Stop and clean up producers
	for _, producer := range resource.NsqProducer {
		if producer != nil {
			producer.Stop()
		}
	}
	resource.NsqProducer = nil

	// Stop and clean up consumer
	if resource.NsqConsumer != nil {
		resource.NsqConsumer.Stop()
		resource.NsqConsumer = nil
	}

	resource.LoggerService.Info("Cleaned up NSQ resources")
}

// InitProducers initializes the NSQ producers using the configuration provided
// in the global config.NsqConfig.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if any producer initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Creates NSQ configuration with timeouts and retry settings
// 2. Initializes producers for each configured NSQ address
// 3. Tests each producer connection with ping
// 4. Stores successfully initialized producers in global resource
func InitProducers(ctx context.Context) error {
	cfg := &config.NsqConfig.NSQ.Producer

	// Create a new NSQ config with proper time unit conversion
	nsqConfig := nsq.NewConfig()
	nsqConfig.DialTimeout = cfg.DialTimeout * time.Second
	nsqConfig.MaxAttempts = uint16(cfg.MaxAttempts)

	// Set additional configuration for better reliability
	nsqConfig.WriteTimeout = 10 * time.Second
	nsqConfig.ReadTimeout = 60 * time.Second      // Increase ReadTimeout to be larger than HeartbeatInterval
	nsqConfig.HeartbeatInterval = 5 * time.Second // Set HeartbeatInterval to be less than ReadTimeout

	resource.LoggerService.Info(fmt.Sprintf("Initializing %d NSQ producers", len(config.NsqConfig.NSQ.Address)))

	var producers []*nsq.Producer
	var mu sync.Mutex

	// Use a wait group to initialize producers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(config.NsqConfig.NSQ.Address))

	for i, addr := range config.NsqConfig.NSQ.Address {
		wg.Add(1)
		go func(index int, address string) {
			defer wg.Done()

			// Create timeout context for this producer initialization
			producerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			resource.LoggerService.Info(fmt.Sprintf("Creating NSQ producer for address: %s", address))

			producer, err := nsq.NewProducer(address, nsqConfig)
			if err != nil {
				resource.LoggerService.Error(fmt.Sprintf("Failed to create producer for address %s: %v", address, err))
				errChan <- fmt.Errorf("failed to create producer for address %s: %w", address, err)
				return
			}

			// Test the connection by sending a ping to the producer
			done := make(chan error, 1)
			go func() {
				done <- producer.Ping()
			}()

			select {
			case err := <-done:
				if err != nil {
					resource.LoggerService.Error(fmt.Sprintf("Producer ping failed for address %s: %v", address, err))
					producer.Stop()
					errChan <- fmt.Errorf("producer ping failed for address %s: %w", address, err)
					return
				}
			case <-producerCtx.Done():
				resource.LoggerService.Error(fmt.Sprintf("Producer ping timeout for address %s", address))
				producer.Stop()
				errChan <- fmt.Errorf("producer ping timeout for address %s", address)
				return
			}

			// Successfully initialized producer
			mu.Lock()
			producers = append(producers, producer)
			mu.Unlock()

			resource.LoggerService.Info(fmt.Sprintf("Successfully initialized NSQ producer for address: %s", address))

		}(i, addr)
	}

	// Wait for all producers to initialize
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		// Clean up any successfully created producers
		for _, p := range producers {
			p.Stop()
		}
		return err
	}

	// Store the successfully initialized producers
	resource.NsqProducer = producers

	resource.LoggerService.Info(fmt.Sprintf("Successfully initialized %d NSQ producers", len(producers)))
	return nil
}

// InitConsumer initializes an NSQ consumer with the specified configuration.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if consumer initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Creates NSQ consumer configuration with proper time unit conversion
// 2. Initializes the consumer with topic and channel
// 3. Validates the consumer configuration
// 4. Stores the consumer in global resource
func InitConsumer(ctx context.Context) error {
	cfg := &config.NsqConfig.NSQ.Consumer

	resource.LoggerService.Info(fmt.Sprintf("Initializing NSQ consumer for topic: %s, channel: %s",
		cfg.Topic, cfg.Channel))

	// Create a new NSQ configuration with proper time unit conversion
	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = cfg.MaxInFlight
	nsqConfig.MaxAttempts = uint16(cfg.MaxAttempts)
	nsqConfig.DefaultRequeueDelay = cfg.RequeueDelay * time.Second
	nsqConfig.HeartbeatInterval = cfg.HeartbeatInterval * time.Second

	// Set additional configuration for better reliability
	nsqConfig.ReadTimeout = 60 * time.Second
	nsqConfig.WriteTimeout = 10 * time.Second
	nsqConfig.DialTimeout = 10 * time.Second

	// Create timeout context for consumer initialization
	consumerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create a new NSQ consumer with the specified topic, channel, and configuration
	done := make(chan struct{})
	var consumer *nsq.Consumer
	var err error

	go func() {
		defer close(done)
		consumer, err = nsq.NewConsumer(cfg.Topic, cfg.Channel, nsqConfig)
	}()

	// Wait for consumer creation or timeout
	select {
	case <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("Failed to create NSQ consumer: %v", err))
			return fmt.Errorf("failed to create NSQ consumer: %w", err)
		}
	case <-consumerCtx.Done():
		resource.LoggerService.Error("NSQ consumer creation timeout")
		return fmt.Errorf("NSQ consumer creation timeout")
	}

	// Validate the created consumer
	if consumer == nil {
		resource.LoggerService.Error("NSQ consumer is nil after creation")
		return fmt.Errorf("NSQ consumer is nil after creation")
	}

	// Store the consumer in global resource
	resource.NsqConsumer = consumer

	resource.LoggerService.Info(fmt.Sprintf("Successfully initialized NSQ consumer for topic: %s, channel: %s",
		cfg.Topic, cfg.Channel))

	return nil
}

// CloseNsq closes all the NSQ connections safely.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if any close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Stops all NSQ producers gracefully with timeout
// 2. Stops the NSQ consumer gracefully with timeout
// 3. Clears all global resource references
func CloseNsq(ctx context.Context) error {
	var errs []error

	// Create timeout context for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Stop all the producers concurrently
	if len(resource.NsqProducer) > 0 {
		resource.LoggerService.Info(fmt.Sprintf("Stopping %d NSQ producers", len(resource.NsqProducer)))

		var wg sync.WaitGroup
		errChan := make(chan error, len(resource.NsqProducer))

		for i, producer := range resource.NsqProducer {
			if producer == nil {
				continue
			}

			wg.Add(1)
			go func(index int, p *nsq.Producer) {
				defer wg.Done()

				done := make(chan struct{})
				go func() {
					defer close(done)
					p.Stop()
				}()

				select {
				case <-done:
					resource.LoggerService.Info(fmt.Sprintf("Successfully stopped NSQ producer %d", index))
				case <-shutdownCtx.Done():
					resource.LoggerService.Error(fmt.Sprintf("Timeout stopping NSQ producer %d", index))
					errChan <- fmt.Errorf("timeout stopping NSQ producer %d", index)
				}
			}(i, producer)
		}

		// Wait for all producers to stop
		wg.Wait()
		close(errChan)

		// Collect any errors
		for err := range errChan {
			errs = append(errs, err)
		}

		// Clear the producers slice
		resource.NsqProducer = nil
		resource.LoggerService.Info("All NSQ producers stopped")
	}

	// Stop the consumer
	if resource.NsqConsumer != nil {
		resource.LoggerService.Info("Stopping NSQ consumer")

		done := make(chan struct{})
		go func() {
			defer close(done)
			resource.NsqConsumer.Stop()
		}()

		select {
		case <-done:
			resource.LoggerService.Info("Successfully stopped NSQ consumer")
		case <-shutdownCtx.Done():
			resource.LoggerService.Error("Timeout stopping NSQ consumer")
			errs = append(errs, fmt.Errorf("timeout stopping NSQ consumer"))
		}

		// Clear the consumer reference
		resource.NsqConsumer = nil
	}

	// Return combined errors if any
	if len(errs) > 0 {
		var combinedErr error
		for _, err := range errs {
			if combinedErr == nil {
				combinedErr = err
			} else {
				combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
			}
		}
		resource.LoggerService.Error(fmt.Sprintf("NSQ shutdown completed with errors: %v", combinedErr))
		return combinedErr
	}

	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 NSQ client shutdown completed successfully")
	}

	return nil
}
