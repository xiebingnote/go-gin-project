package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseMsg
const (
	MsgResponseOK            = "Success"
	HTTPBadRequest           = "Bad Request"
	HTTPNotFound             = "Not Found"
	InvalidParamMessage      = "Invalid Parameters"
	LogicError               = "Logic Error"
	ServerInternalErrMessage = "Internal Error"
)

// Response struct
type Response struct {
	Code      int    `json:"code" binding:"required"`
	Success   bool   `json:"success" binding:"required"`
	ErrMsg    string `json:"message" binding:"required"`
	RequestID string `json:"requestId" binding:"required"`
	Data      any    `json:"result" binding:"required"`
}

// NewOKResp sends a successful HTTP 200 response to the client.
//
// It sets the Content-Type header of the response to "application/json; charset=UTF-8"
// and sends a JSON object with a status code of 200. The response body is populated
// using the OKRestResp function, which includes the provided data and request ID.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values.
//   - data: any, the payload of the response.
//   - reqID: string, the unique identifier for the request.
func NewOKResp(c *gin.Context, data any, reqID string) {
	c.JSON(http.StatusOK, OKRestResp(data, reqID))
}

// OKRestResp returns a pointer to a Response struct with an HTTP status code indicating success.
//
// It sets the Success field to true, indicating that the request was processed successfully.
// The ErrMsg field is an empty string as there is no error. The RequestID is set to the provided
// reqID, allowing for tracking of the request. The Data field holds the response data.
//
// Parameters:
//   - data: any, the payload of the response
//   - reqID: string, the unique identifier for the request
//
// Returns: *Response, a pointer to a Response struct
func OKRestResp(data any, reqID string) *Response {
	return &Response{
		Code:      http.StatusOK, // HTTP 200 OK
		Success:   true,          // Indicates the operation was successful
		ErrMsg:    "",            // No error message
		RequestID: reqID,         // Unique identifier for the request
		Data:      data,          // Payload of the response
	}
}

// NewErrResp returns an error response to the client.
//
// It sets the Content-Type header of the response to "application/json; charset=UTF-8"
// and sets the HTTP status code of the response to the provided statusCode.
// The response body is a JSON object with the ErrMsg field set to errMsg, and the
// RequestID set to reqID.
//
// Parameters:
//   - c: *gin.Context, the Gin context that carries request-scoped values.
//   - statusCode: int, the HTTP status code to set on the response.
//   - errMsg: string, the error message to include in the response body.
//   - reqID: string, the unique identifier for the request.
func NewErrResp(c *gin.Context, statusCode int, errMsg string, reqID string) {
	c.JSON(statusCode, ErrorRestResp(statusCode, errMsg, reqID))
}

// ErrorRestResp returns a *Response with code and success false.
//
// The returned *Response has ErrMsg set to errMsg, and RequestID set to reqID.
// The Data field is set to nil.
//
// Useful for returning an error response to a request.
//
// Example:
//
//	resp := ErrorRestResp(http.StatusBadRequest, "Bad Request", uuid.New().String())
//	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
//	w.WriteHeader(http.StatusBadRequest)
//	if err := json.NewEncoder(w).Encode(resp); err != nil {
//	    panic(err)
//	}
func ErrorRestResp(code int, errMsg string, reqID string) *Response {
	return &Response{
		Code:      code,
		Success:   false,
		ErrMsg:    errMsg,
		RequestID: reqID,
		Data:      nil,
	}
}
