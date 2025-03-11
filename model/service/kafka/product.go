package kafka

import (
	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/IBM/sarama"
)

// SendKafkaMessage sends a message to the specified Kafka topic.
//
// It takes the topic name and the serialized message as arguments.
//
// It returns an error if the message cannot be sent.
func SendKafkaMessage(topic string, producerMessage []byte) error {
	// Create a producer message
	message := &sarama.ProducerMessage{
		Topic: topic,
		// The message value is the serialized message
		Value: sarama.ByteEncoder(producerMessage),
	}

	// Send the message to the topic
	_, _, err := resource.KafkaProducer.SendMessage(message)
	if err != nil {
		return err
	}

	return nil
}

// Producer sends a message to the specified Kafka topic.
//
// It creates a message with the given properties, serializes it to a byte slice,
// and sends it to the topic.
//
// It returns an error if the message cannot be sent.
func Producer() error {
	// Create a message with the given properties
	message := &pkgproto.TestMessage{
		Id:   1,
		Name: "testName",
	}

	// Serialize the message to a byte slice
	serializedData, err := common.SerializeData(message)
	if err != nil {
		return err
	}

	// Send the message to the topic
	if err := SendKafkaMessage(config.KafkaConfig.Kafka.ProducerTopic, serializedData); err != nil {
		return err
	}

	return nil
}
