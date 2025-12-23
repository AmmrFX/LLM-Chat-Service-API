package error

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation_error"
	ErrorTypeTimeout      ErrorType = "timeout_error"
	ErrorTypeLLM          ErrorType = "llm_error"
	ErrorTypeRateLimit    ErrorType = "rate_limit_error"
	ErrorTypeInternal     ErrorType = "internal_error"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized_error"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	StatusCode int       `json:"-"`
	Err        error     `json:"-"`
}

// ------------------------------------------------------------------------------------------------------
// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}



// ------------------------------------------------------------------------------------------------------
// NewValidationError creates a validation error
func NewValidationError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

// ------------------------------------------------------------------------------------------------------
// NewTimeoutError creates a timeout error
func NewTimeoutError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeTimeout,
		Message:    message,
		StatusCode: http.StatusGatewayTimeout,
		Err:        err,
	}
}
// ------------------------------------------------------------------------------------------------------
// NewLLMError creates an LLM API error
func NewLLMError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeLLM,
		Message:    message,
		StatusCode: http.StatusBadGateway,
		Err:        err,
	}
}

// ------------------------------------------------------------------------------------------------------
// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeRateLimit,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
		Err:        err,
	}
}

// ------------------------------------------------------------------------------------------------------
// NewInternalError creates an internal server error
func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// ------------------------------------------------------------------------------------------------------
// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Err:        err,
	}
}

// ------------------------------------------------------------------------------------------------------
// GetHTTPStatusCode returns the appropriate HTTP status code for an error
func GetHTTPStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	// Check for timeout errors
	if errors.Is(err, errors.New("context deadline exceeded")) ||
		errors.Is(err, errors.New("timeout")) {
		return http.StatusGatewayTimeout
	}

	// Default to internal server error
	return http.StatusInternalServerError
}	

// ------------------------------------------------------------------------------------------------------
// ErrorResponse represents the JSON error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ------------------------------------------------------------------------------------------------------
// ErrorDetail contains error details
type ErrorDetail struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
}

// ------------------------------------------------------------------------------------------------------
// NewErrorResponse creates a standardized error response
func NewErrorResponse(err error) ErrorResponse {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return ErrorResponse{
			Error: ErrorDetail{
				Type:    appErr.Type,
				Message: appErr.Message,
				Code:    string(appErr.Type),
			},
		}
	}

	return ErrorResponse{
		Error: ErrorDetail{
			Type:    ErrorTypeInternal,
			Message: err.Error(),
			Code:    string(ErrorTypeInternal),
		},
	}
}
