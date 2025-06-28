package etcd

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

// Init initializes the etcd service by loading the configuration from
// a TOML file and decoding it into the EtcdConfig struct.
//
// It then initializes the etcd service with a background context.
//
// If there is an error getting the current working directory, or the
// etcd configuration file cannot be decoded, the function will panic
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
	if _, err = toml.DecodeFile(rootDir+"/conf/service/etcd.toml", &config.EtcdConfig); err != nil {
		// Panic if the etcd configuration file cannot be decoded
		panic(fmt.Sprintf("Failed to load etcd configuration file: %v", err))
	}

	// Initialize the logger and etcd service with a background context
	service.InitLogger(context.Background())
	service.InitEtcd(context.Background())
}

// TestPutValue tests the PutValue function.
//
// It calls the PutValue function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestPutValue(t *testing.T) {
	err := PutValue()
	if err != nil {
		t.Errorf("%s: %v", "TestPutValue", err)
	} else {
		t.Logf("%s: success", "TestPutValue")
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
