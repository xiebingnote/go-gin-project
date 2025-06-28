package clickhouse

import (
	"context"
	"fmt"
	"log"
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

// Init initializes the ClickHouse service by loading the configuration from
// a TOML file and decoding it into the ClickHouseConfig struct.
//
// It then initializes the logger and ClickHouse service with a background
// context.
//
// If there is an error getting the current working directory, or the
// ClickHouse configuration file cannot be decoded, the function will panic
// with an error message.
func init() {
	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error("Recovered from panic",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			} else {
				// Fallback to standard log if logger is not available
				log.Printf("Test panic recovered: %v\n", r)
				log.Printf("Stack trace: %s\n", string(debug.Stack()))
			}
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

	// Load log configuration from the specified TOML file
	if _, err = toml.DecodeFile(rootDir+"/conf/log/log.toml", &config.LogConfig); err != nil {
		panic("Failed to load log configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/server.toml", &config.ServerConfig); err != nil {
		panic("Failed to load server configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/service/clickhouse.toml", &config.ClickHouseConfig); err != nil {
		// Panic if the ClickHouse configuration file cannot be decoded
		panic(fmt.Sprintf("Failed to load ClickHouse configuration file: %v", err))
	}

	// Initialize the logger and ClickHouse service with a background context
	service.InitLogger(context.Background())
	service.InitClickHouse(context.Background())
}

// TestGetTestAll tests the GetTestAll function.
//
// It calls the GetTestAll function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestGetTestAll(t *testing.T) {
	// Call the GetTestAll function
	res, err := GetTestAll()
	if err != nil {
		// Log an error if there is an error
		t.Errorf("%s: %v", "TestGetTestAll", err)
	} else {
		// Log a success message if the operation is successful
		for _, v := range res {
			fmt.Println("id:", v.Id, "name:", *v.Name, "age:", *v.Age)
		}
		t.Logf("%s: success", "TestGetTestAll")
	}
}
