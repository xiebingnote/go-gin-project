package jwt

import (
	"fmt"
	"net/http"

	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/model/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

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
		// Return an error response if the username or password is empty
		resource.LoggerService.Error(fmt.Sprintf("Registration failed: Username and password are required"))
		resp.NewErrResp(c, http.StatusBadRequest, "Registration failed: Username and password are required", reqID)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Return an error response if password hashing fails
		resource.LoggerService.Error(fmt.Sprintf("Registration failed: Unable to hash password"))
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
		// Return an error response if the username already exists
		resource.LoggerService.Error(fmt.Sprintf("Registration failed: Username already exists"))
		resp.NewErrResp(c, http.StatusConflict, "Registration failed: Username already exists", reqID)
		return
	}

	// Return a success response
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
		resource.LoggerService.Error(fmt.Sprintf("Invalid request: %v", err))
		resp.NewErrResp(c, http.StatusBadRequest, "Invalid request", reqID)
		return
	}

	// Query the user from the database
	var user types.TbUser
	if result := resource.MySQLClient.Table("tb_user").Where("username = ?", login.Username).First(&user); result.Error != nil {
		// Return an error response if the user does not exist
		resource.LoggerService.Error(fmt.Sprintf("Login failed: Invalid credentials"))
		resp.NewErrResp(c, http.StatusUnauthorized, "Invalid credentials", reqID)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		// Return an error response if the password is incorrect
		resource.LoggerService.Error(fmt.Sprintf("Login failed: Invalid credentials"))
		resp.NewErrResp(c, http.StatusUnauthorized, "Invalid credentials", reqID)
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := middleware.GenerateTokenJWT(user.ID)
	if err != nil {
		// Return an error response if JWT token generation fails
		resource.LoggerService.Error(fmt.Sprintf("Login failed: %v", err))
		resp.NewErrResp(c, http.StatusInternalServerError, err.Error(), reqID)
		return
	}

	// Return a success response with the JWT token
	// Return the JWT token
	resp.NewOKResp(c, token, reqID)
}
