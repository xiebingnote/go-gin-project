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

	// Extract the root directory path by splitting on "/pkg"
	dir := strings.Split(rootDir, "/pkg")
	rootDir = dir[0]

	// Load Kafka configuration from the specified TOML file
	if _, err := toml.DecodeFile(rootDir+"/conf/service/kafka.toml", &config.KafkaConfig); err != nil {
		// Panic if the MySQL configuration file cannot be decoded
		panic(fmt.Sprintf("Failed to load Kafka configuration file: %v", err))
	}

	// Initialize the Kafka service with a background context
	service.InitKafka(context.Background())
}

// TestConsumer_Success tests the successful consumption of a message using the Consumer function.
//
// It calls the Consumer function and checks if any error is returned. If an error occurs,
// the test fails with an error message indicating the failure to consume the message.
// Otherwise, it logs a message indicating successful message consumption.
func TestConsumer_Success(t *testing.T) {
	err := Consumer()
	if err != nil {
		t.Errorf("Failed to consumer message: %v", err)
	} else {
		fmt.Println("Message consumer successfully.")
	}
	return
}
