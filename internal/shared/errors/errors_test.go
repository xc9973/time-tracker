package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidationError(t *testing.T) {
	err := ValidationError("category is required")
	if err.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %s", err.Code)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", err.StatusCode)
	}
	if err.Error() != "category is required" {
		t.Errorf("expected message 'category is required', got %s", err.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("session not found")
	if err.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", err.Code)
	}
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", err.StatusCode)
	}
}

func TestConflictError(t *testing.T) {
	session := map[string]interface{}{
		"id":   1,
		"task": "test task",
	}
	err := NewConflictError("session already running", session)
	if err.Code != "CONFLICT" {
		t.Errorf("expected code CONFLICT, got %s", err.Code)
	}
	if err.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", err.StatusCode)
	}
	if err.CurrentSession["id"] != 1 {
		t.Error("expected current session to contain id")
	}
}

func TestRateLimitError(t *testing.T) {
	err := NewRateLimitError(30)
	if err.Code != "RATE_LIMITED" {
		t.Errorf("expected code RATE_LIMITED, got %s", err.Code)
	}
	if err.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", err.StatusCode)
	}
	if err.RetryAfter != 30 {
		t.Errorf("expected retry after 30, got %d", err.RetryAfter)
	}
}

func TestInternalError(t *testing.T) {
	err := InternalError()
	if err.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %s", err.Code)
	}
	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", err.StatusCode)
	}
	// Ensure message doesn't expose internal details
	if err.Message != "An internal error occurred" {
		t.Errorf("expected generic message, got %s", err.Message)
	}
}

func TestWriteError_ValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	err := ValidationError("invalid input")
	WriteError(rr, err)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response ErrorResponse
	json.NewDecoder(rr.Body).Decode(&response)
	if response.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %s", response.Error.Code)
	}
}

func TestWriteError_UnknownError(t *testing.T) {
	rr := httptest.NewRecorder()
	err := errors.New("some internal database error with sensitive info")
	WriteError(rr, err)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}

	var response ErrorResponse
	json.NewDecoder(rr.Body).Decode(&response)
	// Should NOT contain the original error message
	if response.Error.Message != "An internal error occurred" {
		t.Errorf("expected generic message, got %s", response.Error.Message)
	}
}
