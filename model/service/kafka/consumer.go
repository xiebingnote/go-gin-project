package kafka

import (
	"context"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

func DeSerializeData(data []byte) (*pkgproto.TestMessage, error) {
	var message pkgproto.TestMessage
	err := proto.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func Consumer() error {
	for {
		err := resource.KafkaConsumer.Consume(context.Background(), []string{config.KafkaConfig.Kafka.ConsumerTopic}, nil)
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("Kafka consumer error: %v", err))
		}
	}

	return nil
}

func ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Iterate over the messages in the partition
	for message := range claim.Messages() {

		data, err := DeSerializeData(message.Value)
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
