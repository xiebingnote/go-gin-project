package casbin

import (
	"net/http"

	"project/library/middleware"
	"project/library/resource"
	resp "project/library/response"
	"project/model/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Register handles user registration by accepting a JSON request with a username, password, and role,
// hashing the password, and storing the user data in the database. It validates the incoming request
// and ensures the username is unique. If any step fails, it returns an appropriate error response.
// Upon successful registration, it returns a success message.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values, validates the request JSON,
//     and constructs the response.
//
// Behavior:
//   - Binds the incoming JSON request to a struct containing username, password, and role.
//   - Validates the request body and aborts with a 400 Bad Request if invalid.
//   - Hashes the password and aborts with a 500 Internal Server Error if hashing fails.
//   - Creates a new user instance and attempts to insert it into the database.
//   - Aborts with a 409 Conflict if the username already exists.
//   - Responds with a 201 Created and a success message upon successful registration.
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"` // Username chosen by the user
		Password string `json:"password"` // Password chosen by the user
		Role     string `json:"role"`     // Role of the user
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Return an error response if the request body is invalid
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		// Return an error response if password hashing fails
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	user := types.TbUser{
		Username: req.Username,
		Password: string(hashedPassword),
		Role:     req.Role,
	}

	// Insert the user into the database
	if result := resource.MySQLClient.Create(&user); result.Error != nil {
		// Return an error response if the username already exists
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Return a success response
	c.JSON(http.StatusCreated, gin.H{"message": "User created"})
}

// Login authenticates a user by validating the provided username and password,
// generating a JWT token, and returning the token to the client upon successful authentication.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values and manages the request/response lifecycle.
//
// Behavior:
//   - Extracts and validates the JSON request body containing username and password.
//   - Queries the database for a user with the provided username.
//   - Compares the provided password with the stored hashed password.
//   - Generates a JWT token if authentication is successful.
//   - Responds with a 200 OK status and the JWT token if login succeeds.
//   - Returns a 400 Bad Request if the request body is invalid.
//   - Returns a 401 Unauthorized if the username or password is incorrect.
//   - Returns a 500 Internal Server Error if token generation fails.
func Login(c *gin.Context) {
	reqID := uuid.NewString()

	// Define a struct to bind the incoming JSON request
	var req struct {
		Username string `json:"username"` // The username provided by the user
		Password string `json:"password"` // The password provided by the user
	}

	// Bind the incoming JSON request to the struct and validate it
	if err := c.ShouldBindJSON(&req); err != nil {
		// Return an error response if the request body is invalid
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Query the user from the database
	var user types.TbUser
	if result := resource.MySQLClient.Where("username = ?", req.Username).First(&user); result.Error != nil {
		// Return an error response if the user does not exist
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// Return an error response if the password is incorrect
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := middleware.GenerateTokenCasbin(user.ID, user.Role)
	if err != nil {
		// Return an error response if JWT token generation fails
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	// Return the JWT token as a success response
	c.JSON(http.StatusOK, resp.NewOKRestResp(token, reqID))
}
