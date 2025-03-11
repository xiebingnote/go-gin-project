package kafka

import (
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"
)

// Consumer consumes messages from the specified topic and partition.
//
// It will block until the end of the topic/partition is reached.
//
// It will automatically commit the messages it has processed.
//
// Parameters:
//   - None
//
// Returns:
//   - An error if consumer creation fails.
func Consumer() error {
	// Get a partition consumer for the specified topic and partition
	partitionConsumer, err := resource.KafkaConsumer.ConsumePartition(config.KafkaConfig.Kafka.ConsumerTopic, 0, 0)
	if err != nil {
		// Log an error if consumer creation fails
		resource.LoggerService.Error(fmt.Sprintf("Kafka consumer error: %v", err))
		return err
	}
	defer partitionConsumer.Close()

	// Iterate over the messages in the partition
	for msg := range partitionConsumer.Messages() {
		// Deserialize the message
		data, err := common.DeSerializeData(msg.Value, &pkgproto.TestMessage{})
		if err != nil {
			// Log an error if deserialization fails
			resource.LoggerService.Error(fmt.Sprintf("Kafka consumer error: %v", err))
			continue
		}
		// Print the deserialized message
		fmt.Println("data:", data)
		// Log the message
		resource.LoggerService.Info(fmt.Sprintf("Message received: Topic=%s Partition=%d Offset=%d Key=%v Value=%v",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value)))
	}

	return nil
}
