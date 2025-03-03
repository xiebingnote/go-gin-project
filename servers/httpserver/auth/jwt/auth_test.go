package jwt

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/stretchr/testify/suite"
)

// Init initializes the MySQL database connection before the tests are run.
//
// It uses the toml.DecodeFile function to load the MySQL configuration from the
// file located at ./conf/service/mysql.toml. If the file cannot be decoded, it
// panics with the error message.
//
// It then calls service.InitMySQL to establish a connection to the MySQL
// database using the configuration provided.
func init() {
	rootDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir := strings.Split(rootDir, "/servers")
	rootDir = dir[0]

	// Load MySQL configuration
	if _, err := toml.DecodeFile(rootDir+"/conf/service/mysql.toml", &config.MySQLConfig); err != nil {
		// The MySQL configuration file could not be decoded. Panic with the error message.
		panic("Failed to load MySQL configuration file: " + err.Error())
	}

	// Initialize the MySQL database connection
	service.InitMySQL(context.Background())
}

// TestRegister_Success tests the successful registration of a user.
//
// The test creates a test server and a test request, then sends the request to
// the server and asserts that the response status code is 201 Created.
func TestRegister_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/register", Register)

	// Test request
	req, _ := http.NewRequest("POST", "/register", strings.NewReader("username=testuser&password=testpass"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestRegister_Failure tests the failure of registering a user with an empty password.
//
// The test creates a test server and a test request, then sends the request to
// the server and asserts that the response status code is 400 Bad Request.
func TestRegister_Failure(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/register", Register)

	// Test request
	req, _ := http.NewRequest("POST", "/register", strings.NewReader("username=testuser&password="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

type LoginHandlerTestSuite struct {
	suite.Suite
	router *gin.Engine
}

// SetupTest sets up the testing environment for the LoginHandlerTestSuite.
// It initializes the Gin engine in test mode and registers the login route.
func (suite *LoginHandlerTestSuite) SetupTest() {
	// Set Gin to test mode to avoid unwanted outputs during testing.
	gin.SetMode(gin.TestMode)

	// Initialize a new Gin engine for test cases.
	suite.router = gin.Default()

	// Register the login endpoint with the Gin router.
	suite.router.POST("/login", Login)
}

// TestLogin_UserNotFound_ReturnsUnauthorized tests the failure of logging in with a non-existent user.
// The test creates a test server and a test request, then sends the request to the server and asserts
// that the response status code is 401 Unauthorized.
func (suite *LoginHandlerTestSuite) TestLogin_UserNotFound_ReturnsUnauthorized() {
	// Test request
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username":"nonexistent","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid credentials")
}

// TestLogin_PasswordMismatch_ReturnsUnauthorized tests the failure of logging in with an incorrect password.
// The test creates a test server and a test request, then sends the request to the server and asserts
// that the response status code is 401 Unauthorized.
func (suite *LoginHandlerTestSuite) TestLogin_PasswordMismatch_ReturnsUnauthorized() {
	// Test request
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username":"testuser","password":"wrongpassword"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid credentials")
}

// TestLogin_SuccessfulLogin_ReturnsToken tests the successful login of a user.
//
// The test creates a test server and a test request, then sends the request to
// the server and asserts that the response status code is 200 OK. It also
// asserts that the response JSON contains a valid token.
func (suite *LoginHandlerTestSuite) TestLogin_SuccessfulLogin_ReturnsToken() {
	// Create a test request
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(`{"username":"testuser","password":"testpass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Send the request to the server and get the response
	suite.router.ServeHTTP(w, req)

	// Assert that the response status code is 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Assert that the response JSON contains a valid token
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "validtoken", response["token"])
}

// TestLoginHandlerTestSuite is the entry point for running the test suite.
func TestLoginHandlerTestSuite(t *testing.T) {
	// Run the test suite.
	suite.Run(t, new(LoginHandlerTestSuite))
}
