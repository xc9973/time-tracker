// Package errors provides custom error types and error handling for the time tracker.
package errors

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// TimeTrackerError is the base error type for all application errors.
type TimeTrackerError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

// Error implements the error interface.
func (e *TimeTrackerError) Error() string {
	return e.Message
}

// ErrorResponse represents the JSON error response format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error details.
type ErrorDetail struct {
	Code           string                 `json:"code"`
	Message        string                 `json:"message"`
	CurrentSession map[string]interface{} `json:"current_session,omitempty"`
}

// ValidationError represents a 400 Bad Request error for invalid input.
func ValidationError(message string) *TimeTrackerError {
	return &TimeTrackerError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// NotFoundError represents a 404 Not Found error.
func NotFoundError(message string) *TimeTrackerError {
	return &TimeTrackerError{
		Code:       "NOT_FOUND",
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

// ConflictError represents a 409 Conflict error.
type ConflictError struct {
	*TimeTrackerError
	CurrentSession map[string]interface{}
}

// NewConflictError creates a new conflict error with optional current session info.
func NewConflictError(message string, currentSession map[string]interface{}) *ConflictError {
	return &ConflictError{
		TimeTrackerError: &TimeTrackerError{
			Code:       "CONFLICT",
			Message:    message,
			StatusCode: http.StatusConflict,
		},
		CurrentSession: currentSession,
	}
}

// RateLimitError represents a 429 Too Many Requests error.
type RateLimitError struct {
	*TimeTrackerError
	RetryAfter int
}

// NewRateLimitError creates a new rate limit error with retry-after seconds.
func NewRateLimitError(retryAfter int) *RateLimitError {
	return &RateLimitError{
		TimeTrackerError: &TimeTrackerError{
			Code:       "RATE_LIMITED",
			Message:    "Too many requests",
			StatusCode: http.StatusTooManyRequests,
		},
		RetryAfter: retryAfter,
	}
}

// UnauthorizedError represents a 401 Unauthorized error.
func UnauthorizedError(message string) *TimeTrackerError {
	return &TimeTrackerError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// InternalError represents a 500 Internal Server Error.
// Note: This should NOT expose internal details to the client.
func InternalError() *TimeTrackerError {
	return &TimeTrackerError{
		Code:       "INTERNAL_ERROR",
		Message:    "An internal error occurred",
		StatusCode: http.StatusInternalServerError,
	}
}

// WriteError writes an error response to the HTTP response writer.
// It ensures no internal details are exposed in the response.
func WriteError(w http.ResponseWriter, err error) {
	var statusCode int
	var response ErrorResponse

	switch e := err.(type) {
	case *ConflictError:
		statusCode = e.StatusCode
		response = ErrorResponse{
			Error: ErrorDetail{
				Code:           e.Code,
				Message:        e.Message,
				CurrentSession: e.CurrentSession,
			},
		}
	case *RateLimitError:
		statusCode = e.StatusCode
		w.Header().Set("Retry-After", strconv.Itoa(e.RetryAfter))
		response = ErrorResponse{
			Error: ErrorDetail{
				Code:    e.Code,
				Message: e.Message,
			},
		}
	case *TimeTrackerError:
		statusCode = e.StatusCode
		response = ErrorResponse{
			Error: ErrorDetail{
				Code:    e.Code,
				Message: e.Message,
			},
		}
	default:
		// For unknown errors, return a generic internal error
		// to avoid exposing internal details
		statusCode = http.StatusInternalServerError
		response = ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "An internal error occurred",
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
