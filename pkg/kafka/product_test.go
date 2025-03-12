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
		panic("Failed to load Kafka configuration file: " + err.Error())
	}

	// Initialize the Kafka service with a background context
	service.InitKafka(context.Background())
}

// TestProducer_Success tests the successful production of a message.
//
// It calls the Producer function and checks for errors.
//
// If no error occurs, it logs that the message was produced successfully.
//
// If an error occurs, it logs the failure.
func TestProducer_Success(t *testing.T) {
	// Call the Producer function to produce a message
	err := Producer()
	if err != nil {
		// Log an error message if message production fails
		fmt.Println("Failed to produce message:", err)
	} else {
		// Log a success message if message production succeeds
		fmt.Println("Message produced successfully.")
	}
	return
}
