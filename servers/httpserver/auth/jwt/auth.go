package jwt

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/model/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// 常量定义
const (
	MinUsernameLength = 3
	MaxUsernameLength = 20

	MinPasswordLength = 8
	MaxPasswordLength = 128

	ErrUsernameRequired   = "用户名不能为空"
	ErrPasswordRequired   = "密码不能为空"
	ErrUsernameLength     = "用户名长度必须在3-20个字符之间"
	ErrUsernameFormat     = "用户名只能包含字母、数字和下划线"
	ErrPasswordLength     = "密码长度至少8个字符"
	ErrPasswordUppercase  = "密码必须包含至少一个大写字母"
	ErrPasswordLowercase  = "密码必须包含至少一个小写字母"
	ErrPasswordDigit      = "密码必须包含至少一个数字"
	ErrPasswordSpecial    = "密码必须包含至少一个特殊字符"
	ErrInvalidCredentials = "用户名或密码错误"
	ErrUserExists         = "用户名已存在"
	ErrInternalError      = "服务器内部错误"
	ErrInvalidRequest     = "请求格式错误"
)

var (
	// 预编译正则表达式，提高性能
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// ValidationError 验证错误结构
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// validateUsername checks if the provided username meets the requirements.
//
// Requirements:
//  1. The length must be between 3-20 characters.
//  2. It can only contain letters, numbers, and underscores.
//
// Parameters:
//   - username: The username to be validated.
//
// Returns:
//   - An error if the username is invalid.
func validateUsername(username string) error {
	// Check if the length is within the required range
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return ValidationError{
			Field:   "username",
			Message: ErrUsernameLength,
		}
	}

	// Check if the username matches the allowed format
	if !usernameRegex.MatchString(username) {
		return ValidationError{
			Field:   "username",
			Message: ErrUsernameFormat,
		}
	}

	return nil
}

// validatePassword checks if the provided password meets security requirements.
//
// Requirements:
//  1. The length must be between MinPasswordLength and MaxPasswordLength.
//  2. It must contain at least one uppercase letter.
//  3. It must contain at least one lowercase letter.
//  4. It must contain at least one digit.
//  5. It must contain at least one special character.
//
// Parameters:
//   - password: The password to be validated.
//
// Returns:
//   - An error if the password is invalid.
func validatePassword(password string) error {
	// Check the password length
	if len(password) < MinPasswordLength {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordLength,
		}
	}

	if len(password) > MaxPasswordLength {
		return ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("密码长度不能超过%d个字符", MaxPasswordLength),
		}
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)

	// Use the unicode package to accurately check character types
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Check for character type requirements
	if !hasUpper {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordUppercase,
		}
	}

	if !hasLower {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordLowercase,
		}
	}

	if !hasDigit {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordDigit,
		}
	}

	if !hasSpecial {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordSpecial,
		}
	}

	return nil
}

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// validateRequest validates the request data.
//
// It checks for the following:
//  1. Non-empty username and password.
//  2. Username meets the length and format requirements.
//  3. Password meets the length and security requirements.
//
// Parameters:
//   - username: The username to be validated.
//   - password: The password to be validated.
//
// Returns:
//   - An error if the request is invalid.
func validateRequest(username, password string) error {
	// Check for non-empty username and password
	if strings.TrimSpace(username) == "" {
		return ValidationError{
			Field:   "username",
			Message: ErrUsernameRequired,
		}
	}

	if strings.TrimSpace(password) == "" {
		return ValidationError{
			Field:   "password",
			Message: ErrPasswordRequired,
		}
	}

	// Validate username and password
	if err := validateUsername(username); err != nil {
		return err
	}

	if err := validatePassword(password); err != nil {
		return err
	}

	return nil
}

// logAuthEvent 记录认证事件日志
//
// Parameters:
//   - reqID: The request ID.
//   - event: The event name (e.g. "register", "login").
//   - username: The username of the user.
//   - success: Whether the event was successful or not.
//   - err: The error that occurred (if any).
//
// Logs the event with the appropriate log level (INFO for success, ERROR for failure).
// The log message will include the request ID, event name, username, and success indicator.
// If an error occurred, the log message will also include the error details.
func logAuthEvent(reqID, event, username string, success bool, err error) {
	logMsg := fmt.Sprintf("[%s] %s - ", reqID, event)
	logMsg += fmt.Sprintf("用户: %s, ", username)
	logMsg += fmt.Sprintf("成功: %t", success)
	if err != nil {
		logMsg += fmt.Sprintf(", 错误: %v", err)
	}

	if success {
		resource.LoggerService.Info(logMsg)
	} else {
		resource.LoggerService.Error(logMsg)
	}
}

// handleValidationError handles a validation error by logging the error and returning a 400 Bad Request response.
//
// If the error is a ValidationError, it logs the error with the appropriate message and returns a 400 Bad Request response with the error message.
// If the error is not a ValidationError, it logs the error with the appropriate message and returns a 400 Bad Request response with the error message.
func handleValidationError(c *gin.Context, reqID string, err error) {
	// If the error is a ValidationError, log the error with the appropriate message
	if validationErr, ok := err.(ValidationError); ok {
		resource.LoggerService.Error(fmt.Sprintf("[%s] 验证失败: %s", reqID, validationErr.Message))
		// Return a 400 Bad Request response with the error message
		resp.NewErrResp(c, http.StatusBadRequest, validationErr.Message, reqID)
	} else {
		// If the error is not a ValidationError, log the error with the appropriate message
		resource.LoggerService.Error(fmt.Sprintf("[%s] 验证错误: %v", reqID, err))
		// Return a 400 Bad Request response with the error message
		resp.NewErrResp(c, http.StatusBadRequest, err.Error(), reqID)
	}
}

// Register handles user registration by validating the incoming request data,
// hashing the password, and storing the user data in the database. It
// validates the incoming request data and aborts with a 400 Bad Request if
// invalid. It hashes the password and aborts with a 500 Internal Server Error
// if hashing fails. It creates a new user instance and attempts to insert it
// into the database. If the username already exists, it returns a 409 Conflict
// status. Upon successful registration, it returns a 201 Created status with
// the user ID.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values and
//     manages the request/response lifecycle.
//
// Behavior:
//   - Extracts and validates the JSON request body containing username and
//     password.
//   - Hashes the password and aborts with a 500 Internal Server Error if
//     hashing fails.
//   - Creates a new user instance and attempts to insert it into the database.
//   - Aborts with a 409 Conflict if the username already exists.
//   - Responds with a 201 Created and the user ID upon successful registration.
func Register(c *gin.Context) {
	reqID := uuid.NewString()
	startTime := time.Now()

	// Extract and validate the request data
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		logAuthEvent(reqID, "register", "", false, err)
		resp.NewErrResp(c, http.StatusBadRequest, ErrInvalidRequest, reqID)
		return
	}

	// Validate the request data
	if err := validateRequest(req.Username, req.Password); err != nil {
		logAuthEvent(reqID, "register", req.Username, false, err)
		handleValidationError(c, reqID, err)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logAuthEvent(reqID, "register", req.Username, false, err)
		resp.NewErrResp(c, http.StatusInternalServerError, ErrInternalError, reqID)
		return
	}

	// Create a new user instance
	user := types.TbUser{
		Username: req.Username,
		Password: string(hashedPassword),
	}

	// Insert the user into the database
	if result := resource.MySQLClient.Table("tb_user").Create(&user); result.Error != nil {
		// Check for duplicate entry error
		if strings.Contains(result.Error.Error(), "Duplicate entry") {
			logAuthEvent(reqID, "register", req.Username, false, fmt.Errorf("用户名已存在"))
			resp.NewErrResp(c, http.StatusConflict, ErrUserExists, reqID)
			return
		}
		// Handle other database errors
		logAuthEvent(reqID, "register", req.Username, false, result.Error)
		resp.NewErrResp(c, http.StatusInternalServerError, ErrInternalError, reqID)
		return
	}

	// Log successful registration
	duration := time.Since(startTime)
	logAuthEvent(reqID, "register", req.Username, true, nil)
	resource.LoggerService.Info(fmt.Sprintf("[%s] 用户注册成功，耗时: %v", reqID, duration))

	// Return successful response
	resp.NewOKResp(c, gin.H{
		"message": "用户创建成功",
		"user_id": user.ID,
	}, reqID)
}

// Login handles user login by verifying the provided username and password,
// generating a JWT token, and returning it to the client.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values.
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
	startTime := time.Now()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logAuthEvent(reqID, "登录", "", false, err)
		resp.NewErrResp(c, http.StatusBadRequest, ErrInvalidRequest, reqID)
		return
	}

	// Basic validation (excluding password complexity check for login)
	if strings.TrimSpace(req.Username) == "" {
		logAuthEvent(reqID, "登录", "", false, fmt.Errorf("用户名为空"))
		resp.NewErrResp(c, http.StatusBadRequest, ErrUsernameRequired, reqID)
		return
	}

	if strings.TrimSpace(req.Password) == "" {
		logAuthEvent(reqID, "登录", req.Username, false, fmt.Errorf("密码为空"))
		resp.NewErrResp(c, http.StatusBadRequest, ErrPasswordRequired, reqID)
		return
	}

	// Query the user from the database
	var user types.TbUser
	if result := resource.MySQLClient.Table("tb_user").Where("username = ?", req.Username).First(&user); result.Error != nil {
		// Return a uniform error message to prevent username enumeration attacks
		logAuthEvent(reqID, "登录", req.Username, false, fmt.Errorf("用户不存在"))
		resp.NewErrResp(c, http.StatusUnauthorized, ErrInvalidCredentials, reqID)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logAuthEvent(reqID, "登录", req.Username, false, fmt.Errorf("密码错误"))
		resp.NewErrResp(c, http.StatusUnauthorized, ErrInvalidCredentials, reqID)
		return
	}

	// Generate a JWT token for the authenticated user
	token, err := middleware.GenerateTokenJWT(user.ID)
	if err != nil {
		logAuthEvent(reqID, "登录", req.Username, false, err)
		resp.NewErrResp(c, http.StatusInternalServerError, ErrInternalError, reqID)
		return
	}

	// Log successful login
	duration := time.Since(startTime)
	logAuthEvent(reqID, "登录", req.Username, true, nil)
	resource.LoggerService.Info(fmt.Sprintf("[%s] 用户登录成功，耗时: %v", reqID, duration))

	// Return a successful response containing the JWT token
	resp.NewOKResp(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
		"message":  "登录成功",
	}, reqID)
}
