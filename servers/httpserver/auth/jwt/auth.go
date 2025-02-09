package jwt

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
	// Extract form data
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Return an error response if password hashing fails
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": "Registration failed: Unable to hash password"},
		)
		return
	}

	// Create a new user instance
	user := types.TbUser{
		Username: username,
		Password: string(hashedPassword),
	}

	// Insert the user into the database
	if result := resource.MySQLClient.Create(&user); result.Error != nil {
		// Return an error response if the username already exists
		c.AbortWithStatusJSON(
			http.StatusConflict,
			gin.H{"error": "Registration failed: Username already exists"},
		)
		return
	}

	// Return a success response
	c.Status(http.StatusCreated)
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

	// Extract form data for username and password
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Query the user from the database
	var user types.TbUser
	if result := resource.MySQLClient.Where("username = ?", username).First(&user); result.Error != nil {
		// Return an error response if the user does not exist
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// Return an error response if the password is incorrect
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := middleware.GenerateTokenJWT(user.ID)
	if err != nil {
		// Return an error response if JWT token generation fails
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	// Return a success response with the JWT token
	// Return the JWT token
	c.JSON(http.StatusOK, resp.NewOKRestResp(token, reqID))
}
