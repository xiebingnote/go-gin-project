package dao

import (
	"context"
	"github.com/BurntSushi/toml"
	"go-gin-project/bootstrap/service"
	"go-gin-project/library/config"
	"os"
	"strings"
	"testing"
)

var userClient *UserClient

// Init initializes the user client and loads necessary configurations.
//
// It retrieves the current working directory, loads the MySQL configuration
// from a TOML file, and initializes the MySQL service. It then creates a new
// instance of UserClient.
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

	// Load MySQL configuration from the specified TOML file
	if _, err := toml.DecodeFile(rootDir+"/conf/service/mysql.toml", &config.MySQLConfig); err != nil {
		// Panic if the MySQL configuration file cannot be decoded
		panic("Failed to load MySQL configuration file: " + err.Error())
	}

	// Initialize the MySQL service with a background context
	service.InitMySQL(context.Background())

	// Create a new UserClient instance and assign it to the global variable
	userClient = NewUserClient()
}

// TestCreateDb tests the creation of the database table.
//
// It calls the CreateTb method on the userClient to create the database table.
// If the table creation fails, the test will log an error.
func TestCreateDb(t *testing.T) {
	// Attempt to create the database table
	err := userClient.CreateTb()
	if err != nil {
		// Log an error if the table creation fails
		t.Error(err)
	}
}

// TestGetUserNameByID tests the retrieval of a username by ID.
//
// It calls the GetUserNameByID method on the userClient to retrieve the username
// associated with the given ID. If the retrieval fails, the test will log an error.
func TestGetUserNameByID(t *testing.T) {
	id := "1"
	name, err := userClient.GetUserNameByID(id)
	if err != nil {
		// Log an error if the retrieval fails
		t.Error(err)
	}
	t.Logf("Username for ID %s is %s", id, name)
}
