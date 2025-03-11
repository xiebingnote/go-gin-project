package kafka

import (
	"context"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/IBM/sarama"
)

// ExampleConsumerGroupHandler is a consumer group handler that implements
type ExampleConsumerGroupHandler struct {
	Ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
// It closes the Ready channel to signal that the consumer is ready.
func (h *ExampleConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	// Close the Ready channel to indicate readiness
	close(h.Ready)
	return nil
}

// Cleanup is called once all ConsumeClaim goroutines have exited.
// It is a last chance to clean up any resources, but it is not
// a guarantee that it will be called in all cases (e.g., if the
// process is killed).
func (h *ExampleConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim is called once for each consumer claim being consumed.
// A claim is a unique partition of a topic that the consumer is responsible
// for consuming. The claim is closed when the consumer is done consuming.
//
// This function will be called for each message in the partition until the
// message queue is empty, at which point the claim will be closed.
func (h *ExampleConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Iterate over the messages in the partition
	for message := range claim.Messages() {
		data, err := common.DeSerializeData(message.Value, &pkgproto.TestMessage{})
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("Kafka consumer error: %v", err))
			continue
		}
		fmt.Println(data)

		// Log the message
		resource.LoggerService.Info(fmt.Sprintf("Message received: Topic=%s Partition=%d Offset=%d Key=%s Value=%s",
			message.Topic, message.Partition, message.Offset,
			string(message.Key), string(message.Value)))

		// Mark the message as processed
		session.MarkMessage(message, "")
	}
	return nil
}

// StartKafkaConsumer starts a Kafka consumer in a separate goroutine,
// consuming messages from the given topics with the given handler.
//
// It waits until the consumer is ready, and returns an error if the consumer fails
// to start.
//
// Parameters:
//
//	topics: The topics to consume from.
//	handler: The handler that will process the messages.
//
// Returns:
//
//	An error if the consumer fails to start.
func StartKafkaConsumer(topics []string, handler *ExampleConsumerGroupHandler) error {
	// Start the consumer in a separate goroutine
	go func() {
		for {
			// Consume messages from the specified topics
			if err := resource.KafkaConsumerGroup.Consume(context.Background(), topics, handler); err != nil {
				resource.LoggerService.Error(fmt.Sprintf("Kafka consumer error: %v", err))
				return
			}
		}
	}()

	//Wait until the consumer is ready
	<-handler.Ready

	return nil
}
