package casbin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Init initializes the MySQL database connection for testing.
//
// It retrieves the current working directory, loads the MySQL configuration
// from a TOML file, and initializes the MySQL service. If any step fails,
// it will panic with an error message.
func init() {
	// Retrieve the current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		// Panic if there is an error getting the working directory
		panic(err)
	}

	// Extract the root directory path by splitting on "/servers"
	dir := strings.Split(rootDir, "/servers")
	rootDir = dir[0]

	// Load MySQL configuration from the specified TOML file
	if _, err := toml.DecodeFile(rootDir+"/conf/service/mysql.toml", &config.MySQLConfig); err != nil {
		// Panic if the MySQL configuration file cannot be decoded
		panic("Failed to load MySQL configuration file: " + err.Error())
	}

	// Initialize the MySQL service with a background context
	service.InitMySQL(context.Background())
}

// TestRegister_InvalidJSONRequest_ReturnsBadRequest tests that the Register
// handler returns an HTTP 400 Bad Request when the request body contains
// invalid JSON.
//
// The test creates a test server and a test request with invalid JSON,
// then sends the request to the server and asserts that the response status
// code is 400 Bad Request.
func TestRegister_InvalidJSONRequest_ReturnsBadRequest(t *testing.T) {
	// Set the Gin mode to test mode
	gin.SetMode(gin.TestMode)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Create a test request with invalid JSON
	c.Request, _ = http.NewRequest("POST", "/register", bytes.NewBufferString(`{"username": "testuser", "password": "testpass"`))

	// Call the Register handler
	Register(c)

	// Assert that the response status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRegister_MissingUsernameOrPassword_ReturnsBadRequest tests that the
// Register handler returns an HTTP 400 Bad Request when the request body
// contains either a missing username or password.
//
// The test creates a test server and a test request with missing username
// and password, then sends the request to the server and asserts that the
// response status code is 400 Bad Request.
func TestRegister_MissingUsernameOrPassword_ReturnsBadRequest(t *testing.T) {
	// Set the Gin mode to test mode
	gin.SetMode(gin.TestMode)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Create a test request with missing username
	c.Request, _ = http.NewRequest("POST", "/register", bytes.NewBufferString(`{"password": "testpass", "role": "user"}`))

	// Call the Register handler
	Register(c)

	// Assert that the response status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRegister_UsernameConflict_ReturnsConflict tests that the Register
// handler returns an HTTP 409 Conflict when attempting to register with
// an existing username.
//
// The test sets up a test server and a test request with a username that
// already exists in the database, sends the request to the server, and
// asserts that the response status code is 409 Conflict.
func TestRegister_UsernameConflict_ReturnsConflict(t *testing.T) {
	// Set Gin mode to test mode
	gin.SetMode(gin.TestMode)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Create a test request with an existing username
	c.Request, _ = http.NewRequest("POST", "/register", bytes.NewBufferString(`{"username": "existinguser", "password": "testpass", "role": "user"}`))

	// Call the Register handler
	Register(c)

	// Assert that the response status code is 409 Conflict
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestRegister_SuccessfulRegistration_ReturnsOK tests that the Register
// handler returns an HTTP 200 OK response when registration is successful.
//
// The test creates a test server and a test request with valid registration
// details, sends the request to the server, and asserts that the response
// status code is 200 OK.
func TestRegister_SuccessfulRegistration_ReturnsOK(t *testing.T) {
	// Set the Gin mode to test mode
	gin.SetMode(gin.TestMode)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Create a test request with valid registration details
	c.Request, _ = http.NewRequest("POST", "/register", bytes.NewBufferString(`{"username": "user1", "password": "testpass", "role": "user"}`))

	// Call the Register handler
	Register(c)

	// Assert that the response status code is 200 OK
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestLogin_InvalidJSONRequest_ReturnsBadRequest verifies that the Login handler
// returns an HTTP 400 Bad Request when the request body contains invalid JSON.
//
// The test creates a test server and a test request with an invalid JSON
// payload, then sends the request to the server and asserts that the response
// status code is 400 Bad Request.
func TestLogin_InvalidJSONRequest_ReturnsBadRequest(t *testing.T) {
	// Set the Gin mode to test mode
	gin.SetMode(gin.TestMode)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Create a test request with invalid JSON
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call the Login handler
	Login(c)

	// Assert that the response status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestLogin_MissingUsernameOrPassword_ReturnsBadRequest tests that the Login
// handler returns an HTTP 400 Bad Request when the request body contains
// either a missing username or password.
//
// The test creates a test server and a test request with missing username,
// then sends the request to the server and asserts that the response status
// code is 400 Bad Request.
func TestLogin_MissingUsernameOrPassword_ReturnsBadRequest(t *testing.T) {
	// Arrange: Create a test response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request with missing username
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username": ""}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// Act: Call the Login handler
	Login(c)

	// Assert: Verify that the response status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestLogin_UserNotFound_ReturnsUnauthorized tests that the Login handler
// returns an HTTP 401 Unauthorized response when a user is not found in the
// database.
//
// The test creates a test server and a test request with a valid username
// and password, then sends the request to the server and asserts that the
// response status code is 401 Unauthorized.
func TestLogin_UserNotFound_ReturnsUnauthorized(t *testing.T) {
	// Arrange: Create a test response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request with a valid username and password
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username": "user1", "password": "testpass"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// Act: Call the Login handler
	Login(c)

	// Assert: Verify that the response status code is 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestLogin_PasswordMismatch_ReturnsUnauthorized verifies that the Login handler
// returns an HTTP 401 Unauthorized when the password is incorrect.
//
// The test creates a simulated HTTP request with a mismatched password and
// sends it to the server. It then checks that the server responds with a
// 401 Unauthorized status code.
func TestLogin_PasswordMismatch_ReturnsUnauthorized(t *testing.T) {
	// Arrange: Create a test response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request with an incorrect password
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username": "user1", "password": "wrongpass"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// Act: Call the Login handler
	Login(c)

	// Assert: Verify that the response status code is 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestLogin_SuccessfulLogin_ReturnsOK tests that the Login handler
// returns an HTTP 200 OK status when the login is successful.
//
// The test creates a test server and a test request with valid login
// credentials, sends the request to the server, and asserts that the
// response status code is 200 OK.
func TestLogin_SuccessfulLogin_ReturnsOK(t *testing.T) {
	// Arrange: Create a test response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request with valid username and password
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username": "user1", "password": "testpass"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// Act: Call the Login handler
	Login(c)

	// Assert: Verify that the response status code is 200 OK
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}
