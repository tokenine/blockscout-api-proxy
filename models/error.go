package models

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// NewErrorResponse creates a new ErrorResponse
func NewErrorResponse(error, message string) *ErrorResponse {
	return &ErrorResponse{
		Error:   error,
		Message: message,
	}
}