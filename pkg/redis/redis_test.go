package redis

import (
	"context"
	"fmt"
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

// Init initializes the Redis service by loading the configuration from a TOML
// file and decoding it into the RedisConfig struct.
//
// It then initializes the Redis service with a background context.
//
// If there is an error getting the current working directory, or the Redis
// configuration file cannot be decoded, the function will panic with an error
// message.
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

	// Load log configuration from the specified TOML file
	if _, err = toml.DecodeFile(rootDir+"/conf/log/log.toml", &config.LogConfig); err != nil {
		panic(fmt.Sprintf("Failed to load log configuration file: %v", err))
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/server.toml", &config.ServerConfig); err != nil {
		panic(fmt.Sprintf("Failed to load server configuration file: %v", err))
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/service/redis.toml", &config.RedisConfig); err != nil {
		// Panic if the Redis configuration file cannot be decoded
		panic(fmt.Sprintf("Failed to load Redis configuration file: %v", err))
	}

	// Initialize the logger and Redis service with a background context
	service.InitLogger(context.Background())
	service.InitRedis(context.Background())
}

// TestSetValue tests the SetValue function.
//
// It calls the SetValue function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestSetValue(t *testing.T) {
	err := SetValue()
	if err != nil {
		t.Errorf("%s: %v", "TestSetValue", err)
	} else {
		t.Logf("%s: success", "TestSetValue")
	}
}

// TestGetValue tests the GetValue function.
//
// It calls the GetValue function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestGetValue(t *testing.T) {
	err := GetValue()
	if err != nil {
		t.Errorf("%s: %v", "TestGetValue", err)
	} else {
		t.Logf("%s: success", "TestGetValue")
	}
}

// TestListValue tests the ListValue function.
//
// It calls the ListValue function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestListValue(t *testing.T) {
	err := ListValue()
	if err != nil {
		t.Errorf("%s: %v", "TestListValue", err)
	} else {
		t.Logf("%s: success", "TestListValue")
	}
}

// TestHashValue tests the HashValue function.
//
// It calls the HashValue function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestHashValue(t *testing.T) {
	err := HashValue()
	if err != nil {
		t.Errorf("%s: %v", "TestHashValue", err)
	} else {
		t.Logf("%s: success", "TestHashValue")
	}
}
