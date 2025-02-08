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
// hashing the password, and storing the user information in the database. It returns
// an error response if any step fails, or a success response upon successful registration.
func Register(c *gin.Context) {
	// Extract form data
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Return an error response if password hashing fails
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
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
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Return a success response
	c.Status(http.StatusCreated)
}

// Login handles user login by verifying the provided username and password,
// generating a JWT token, and returning the token to the client.
//
// It returns an error response if any step fails, or a success response with a
// JWT token upon successful login.
func Login(c *gin.Context) {
	reqID := uuid.NewString()

	// Get form data
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Query the user
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

	// Generate a JWT token
	token, err := middleware.GenerateTokenJWT(user.ID)
	if err != nil {
		// Return an error response if JWT token generation fails
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	// Return the JWT token
	c.JSON(http.StatusOK, resp.NewOKRestResp(token, reqID))
}
