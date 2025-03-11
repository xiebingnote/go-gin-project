package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"

	"github.com/BurntSushi/toml"
)

// Init initializes the Kafka service by loading the configuration from
// a TOML file and decoding it into the KafkaConfig struct.
//
// It then initializes the Kafka service with a background context.
//
// If there is an error getting the current working directory, or the
// Kafka configuration file cannot be decoded, the function will panic
// with an error message.
func init() {
	// Retrieve the current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		// Panic if there is an error getting the working directory
		panic(err)
	}

	// Extract the root directory path by splitting on "/model"
	dir := strings.Split(rootDir, "/model")
	rootDir = dir[0]

	// Load Kafka configuration from the specified TOML file
	if _, err := toml.DecodeFile(rootDir+"/conf/service/kafka.toml", &config.KafkaConfig); err != nil {
		// Panic if the MySQL configuration file cannot be decoded
		panic("Failed to load Kafka configuration file: " + err.Error())
	}

	// Initialize the Kafka service with a background context
	service.InitKafka(context.Background())
}

// TestConsumerGroup_Success tests the successful operation of a Kafka consumer group.
//
// It initializes a consumer group handler and starts the Kafka consumer with the
// specified topic and handler. It then checks for errors, logging an error if the
// consumer fails to start, otherwise logging a success message.
func TestConsumerGroup_Success(t *testing.T) {
	// Initialize the consumer group handler with a ready channel
	handler := &ExampleConsumerGroupHandler{
		Ready: make(chan bool),
	}

	// Start the Kafka consumer with the specified topic and handler
	err := StartKafkaConsumer(config.KafkaConfig.Kafka.ConsumerGroupTopic, handler)
	if err != nil {
		// Log an error if the consumer fails to start
		t.Errorf("Failed to consumer message: %v", err)
	} else {
		// Log a success message if the consumer starts successfully
		fmt.Println("Message consumer successfully.")
	}
}
