package response

import "net/http"

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

// NewOKRestResp returns a pointer to a Response struct with an HTTP status code indicating success.
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
func NewOKRestResp(data any, reqID string) *Response {
	return &Response{
		Code:      http.StatusOK, // HTTP 200 OK
		Success:   true,          // Indicates the operation was successful
		ErrMsg:    "",            // No error message
		RequestID: reqID,         // Unique identifier for the request
		Data:      data,          // Payload of the response
	}
}

// NewErrorRestResp returns a *Response with code and success false.
//
// The returned *Response has ErrMsg set to errMsg, and RequestID set to reqID.
// The Data field is set to nil.
//
// Useful for returning an error response to a request.
//
// Example:
//
//	resp := NewErrorRestResp(http.StatusBadRequest, "Bad Request", uuid.New().String())
//	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
//	w.WriteHeader(http.StatusBadRequest)
//	if err := json.NewEncoder(w).Encode(resp); err != nil {
//	    panic(err)
//	}
func NewErrorRestResp(code int, errMsg string, reqID string) *Response {
	return &Response{
		Code:      code,
		Success:   false,
		ErrMsg:    errMsg,
		RequestID: reqID,
		Data:      nil,
	}
}

// NewUnAuthorizedRestResp returns a *Response with code, success false, and ErrMsg set to errMsg.
//
// The returned *Response has RequestID set to reqID, and Data set to data.
//
// Useful for returning an unauthorized response to a request.
//
// Example:
//
//	resp := NewUnAuthorizedRestResp(http.StatusUnauthorized, "Bad credentials", uuid.New().String())
//	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
//	w.WriteHeader(http.StatusUnauthorized)
//	if err := json.NewEncoder(w).Encode(resp); err != nil {
//	    panic(err)
//	}
func NewUnAuthorizedRestResp(code int, errMsg string, reqID string) *Response {
	return &Response{
		Code:      code,
		Success:   false,
		ErrMsg:    errMsg,
		RequestID: reqID,
		Data:      nil,
	}
}
