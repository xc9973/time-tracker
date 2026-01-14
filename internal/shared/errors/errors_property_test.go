package errors

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: time-tracker, Property 14: 错误响应安全性
// *For any* 错误响应，不应包含内部系统细节、堆栈跟踪或敏感配置信息。
// **Validates: Requirements 4.14**

func TestErrorResponse_Property14_NoInternalDetails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random internal error messages that might contain sensitive info
		sensitivePatterns := []string{
			"database connection failed: %s",
			"SQL error: %s",
			"file not found: %s",
			"panic: %s",
			"stack trace: %s",
			"config error: API_KEY=%s",
			"internal server error: %s",
		}
		pattern := rapid.SampledFrom(sensitivePatterns).Draw(t, "pattern")
		sensitiveData := rapid.StringMatching(`[a-zA-Z0-9_/]{10,50}`).Draw(t, "sensitiveData")
		internalError := errors.New(strings.Replace(pattern, "%s", sensitiveData, 1))

		rr := httptest.NewRecorder()
		WriteError(rr, internalError)

		// Parse the response
		var response ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Response should NOT contain the sensitive data
		responseStr := response.Error.Message
		if strings.Contains(responseStr, sensitiveData) {
			t.Fatalf("error response contains sensitive data: %s", responseStr)
		}

		// Response should be a generic message
		if response.Error.Message != "An internal error occurred" {
			t.Fatalf("expected generic message, got: %s", response.Error.Message)
		}

		// Response code should be INTERNAL_ERROR
		if response.Error.Code != "INTERNAL_ERROR" {
			t.Fatalf("expected code INTERNAL_ERROR, got: %s", response.Error.Code)
		}
	})
}

func TestErrorResponse_Property14_KnownErrorsPreserveMessage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user-facing error messages (safe to expose)
		message := rapid.StringMatching(`[a-zA-Z ]{5,50}`).Draw(t, "message")

		// Test with ValidationError
		rr := httptest.NewRecorder()
		WriteError(rr, ValidationError(message))

		var response ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Known errors should preserve their message
		if response.Error.Message != message {
			t.Fatalf("expected message %q, got %q", message, response.Error.Message)
		}
	})
}

func TestErrorResponse_Property14_NoStackTraces(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate error messages that look like stack traces
		stackPatterns := []string{
			"goroutine 1 [running]:\nmain.main()\n\t/app/main.go:10",
			"panic: runtime error: index out of range\ngoroutine 1",
			"at Object.<anonymous> (/app/server.js:42:15)",
			"Traceback (most recent call last):\n  File \"/app/main.py\"",
		}
		stackTrace := rapid.SampledFrom(stackPatterns).Draw(t, "stackTrace")
		internalError := errors.New(stackTrace)

		rr := httptest.NewRecorder()
		WriteError(rr, internalError)

		var response ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Response should NOT contain stack trace patterns
		dangerousPatterns := []string{"goroutine", "panic:", "Traceback", ".go:", ".js:", ".py:"}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(response.Error.Message, pattern) {
				t.Fatalf("error response contains stack trace pattern %q: %s", pattern, response.Error.Message)
			}
		}
	})
}
