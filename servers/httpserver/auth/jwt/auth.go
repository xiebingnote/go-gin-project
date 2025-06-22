package jwt

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/model/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// validateUsername checks if the username meets the requirements
func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 20 {
		return fmt.Errorf("username must be between 3 and 20 characters")
	}

	// Only allow letters, numbers, and underscores
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	if err != nil || !matched {
		return fmt.Errorf("username can only contain letters, numbers, and underscores")
	}

	return nil
}

// validatePassword checks if the password meets the requirements
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Check for at least one uppercase letter
	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	// Check for at least one lowercase letter
	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	// Check for at least one number
	if !strings.ContainsAny(password, "0123456789") {
		return fmt.Errorf("password must contain at least one number")
	}

	// Check for at least one special character
	if !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// Register handles user registration by processing the provided username and password,
// hashing the password, and storing the user information in the database.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values.
//
// Behavior:
//   - Extracts and validates the incoming request form data.
//   - Hashes the password using bcrypt.
//   - Creates a new user instance and attempts to insert it into the database.
//   - Aborts with a 409 Conflict if the username already exists.
//   - Returns a 201 Created response upon successful registration.
func Register(c *gin.Context) {
	reqID := uuid.NewString()
	// Extract form data
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Username and password are required", reqID))
		resp.NewErrResp(c, http.StatusBadRequest, "Registration failed: Username and password are required", reqID)
		return
	}

	// Validate username
	if err := validateUsername(username); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Username validation error: %v", reqID, err))
		resp.NewErrResp(c, http.StatusBadRequest, fmt.Sprintf("Registration failed: %v", err), reqID)
		return
	}

	// Validate password
	if err := validatePassword(password); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Password validation error: %v", reqID, err))
		resp.NewErrResp(c, http.StatusBadRequest, fmt.Sprintf("Registration failed: %v", err), reqID)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Password hashing error: %v", reqID, err))
		resp.NewErrResp(c, http.StatusInternalServerError, "Registration failed: Unable to hash password", reqID)
		return
	}

	// Create a new user instance
	user := types.TbUser{
		Username: username,
		Password: string(hashedPassword),
	}

	// Insert the user into the database
	if result := resource.MySQLClient.Table("tb_user").Create(&user); result.Error != nil {
		// Check for duplicate entry error
		if strings.Contains(result.Error.Error(), "Duplicate entry") {
			resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Username '%s' already exists", reqID, username))
			resp.NewErrResp(c, http.StatusConflict, "Registration failed: Username already exists", reqID)
			return
		}
		// Handle other database errors
		resource.LoggerService.Error(fmt.Sprintf("[%s] Registration failed: Database error: %v", reqID, result.Error))
		resp.NewErrResp(c, http.StatusInternalServerError, "Registration failed: Database error", reqID)
		return
	}

	// Return a success response
	resource.LoggerService.Info(fmt.Sprintf("[%s] User '%s' registered successfully", reqID, username))
	resp.NewOKResp(c, "User created", reqID)
}

type LoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles user login by verifying the provided username and password,
// generating a JWT token, and returning the token to the client.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values.
//
// Behavior:
//   - Extracts form data for username and password.
//   - Queries the database for a user with the provided username.
//   - Verifies the provided password against the stored hashed password.
//   - Generates a JWT token if authentication is successful.
//   - Responds with a 200 OK status and the JWT token if login succeeds.
//   - Returns a 401 Unauthorized if the username or password is incorrect.
//   - Returns a 500 Internal Server Error if token generation fails.
func Login(c *gin.Context) {
	reqID := uuid.NewString()

	var login LoginInfo

	if err := c.BindJSON(&login); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Login failed: Invalid request format: %v", reqID, err))
		resp.NewErrResp(c, http.StatusBadRequest, "Invalid request format", reqID)
		return
	}

	// Validate username
	if err := validateUsername(login.Username); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Login failed: Username validation error: %v", reqID, err))
		resp.NewErrResp(c, http.StatusBadRequest, fmt.Sprintf("Login failed: %v", err), reqID)
		return
	}

	// Query the user from the database
	var user types.TbUser
	if result := resource.MySQLClient.Table("tb_user").Where("username = ?", login.Username).First(&user); result.Error != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Login failed: User '%s' not found: %v", reqID, login.Username, result.Error))
		resp.NewErrResp(c, http.StatusUnauthorized, "Invalid credentials", reqID)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Login failed: Invalid password for user '%s'", reqID, login.Username))
		resp.NewErrResp(c, http.StatusUnauthorized, "Invalid credentials", reqID)
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := middleware.GenerateTokenJWT(user.ID)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("[%s] Login failed: Token generation error for user '%s': %v", reqID, login.Username, err))
		resp.NewErrResp(c, http.StatusInternalServerError, "Login failed: Unable to generate token", reqID)
		return
	}

	// Return a success response with the JWT token
	resource.LoggerService.Info(fmt.Sprintf("[%s] User '%s' logged in successfully", reqID, login.Username))
	resp.NewOKResp(c, token, reqID)
}
