package nsq

import (
	"context"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
)

// Init initializes the Nsq service by loading the configuration from
// a TOML file and decoding it into the NsqConfig struct.
//
// It then initializes the Nsq service with a background context.
//
// If there is an error getting the current working directory, or the
// Nsq configuration file, cannot be decoded, the function will panic
// with an error message.
func init() {
	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			resource.LoggerService.Error("Recovered from panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
			)
			// Print the stack trace
			debug.PrintStack()
		}
	}()

	// Retrieve the current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		// Panic if there is an error getting the working directory
		panic(err)
	}

	// Extract the root directory path by splitting on "/pkg"
	dir := strings.Split(rootDir, "/pkg")
	rootDir = dir[0]

	// Load configuration from the specified TOML file
	if _, err = toml.DecodeFile(rootDir+"/conf/log/log.toml", &config.LogConfig); err != nil {
		panic("Failed to load log configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/server.toml", &config.ServerConfig); err != nil {
		panic("Failed to load server configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/service/nsq.toml", &config.NsqConfig); err != nil {
		panic("Failed to load nsq configuration file: " + err.Error())
	}

	// Initialize the logger and Nsq service with a background context
	service.InitLogger(context.Background())
	service.InitNSQ(context.Background())
}

// TestProducer_Success tests the successful production of a message.
//
// It calls the Producer function and checks for errors. If an error occurs,
// it logs an error message indicating the failure to produce the message.
// Otherwise, it logs a success message.
func TestProducer_Success(t *testing.T) {
	// Call the Producer function to produce a message
	err := Producer()
	if err != nil {
		// Log an error message if message production fails
		t.Errorf("Failed to produce message: %v", err)
	} else {
		// Log a success message if message production succeeds
		t.Logf("Message produced successfully.")
	}
	return
}
