package elasticsearch

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"go.uber.org/zap"
	"os"
	"runtime/debug"
	"strings"
	"testing"
)

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
		panic("Failed to load log configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/server.toml", &config.ServerConfig); err != nil {
		panic("Failed to load server configuration file: " + err.Error())
	}
	if _, err = toml.DecodeFile(rootDir+"/conf/service/elasticsearch.toml", &config.ElasticSearchConfig); err != nil {
		// Panic if the Elasticsearch configuration file cannot be decoded
		panic(fmt.Sprintf("Failed to load Elasticsearch configuration file: %v", err))
	}

	// Initialize the logger and Elasticsearch service with a background context
	service.InitLogger(context.Background())
	service.InitElasticSearch(context.Background())
}

// TestInsert tests the Insert function.
//
// It calls the Insert function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestInsert(t *testing.T) {
	err := Insert()
	if err != nil {
		// Log an error if there is an error
		t.Errorf("%s: %v", "TestInsert", err)
	} else {
		// Log a success message if the operation is successful
		t.Logf("%s: success", "TestInsert")
	}
}

// TestSearch tests the Search function.
//
// It calls the Search function and checks for any errors. If an error
// occurs, it logs an error message with the test name and the error.
// Otherwise, it logs a success message with the test name.
func TestSearch(t *testing.T) {
	// Call the Search function to execute the search operation
	err := Search()
	if err != nil {
		// Log an error if there is an error
		t.Errorf("%s: %v", "TestSearch", err)
	} else {
		// Log a success message if the operation is successful
		t.Logf("%s: success", "TestSearch")
	}
}
