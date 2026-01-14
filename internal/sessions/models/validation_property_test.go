package models

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestValidation_Property13_SessionStartSecurity tests that SessionStart
// also handles special characters correctly.
func TestValidation_Property13_SessionStartSecurity(t *testing.T) {
	maliciousInputs := []string{
		"'; DROP TABLE sessions; --",
		"<script>alert('XSS')</script>",
		"${7*7}",
		"{{constructor.constructor('return this')()}}",
		"../../../etc/passwd",
		"file:///etc/passwd",
		"data:text/html,<script>alert('XSS')</script>",
	}

	rapid.Check(t, func(t *rapid.T) {
		malicious := rapid.SampledFrom(maliciousInputs).Draw(t, "malicious")

		session := &SessionStart{
			Category: "test",
			Task:     malicious,
		}

		err := session.Validate()
		if err != nil {
			t.Fatalf("validation should pass for malicious input, got: %v", err)
		}

		// Content should be preserved
		expected := strings.TrimSpace(malicious)
		if session.Task != expected {
			t.Fatalf("malicious input not preserved: expected %q, got %q", expected, session.Task)
		}
	})
}

// TestValidation_Property13_SessionStopSecurity tests that SessionStop
// also handles special characters correctly.
func TestValidation_Property13_SessionStopSecurity(t *testing.T) {
	maliciousInputs := []string{
		"'; DROP TABLE sessions; --",
		"<script>alert('XSS')</script>",
		"${7*7}",
		"{{constructor.constructor('return this')()}}",
	}

	rapid.Check(t, func(t *rapid.T) {
		malicious := rapid.SampledFrom(maliciousInputs).Draw(t, "malicious")

		session := &SessionStop{
			Note: &malicious,
		}

		err := session.Validate()
		if err != nil {
			t.Fatalf("validation should pass for malicious input, got: %v", err)
		}

		// Content should be preserved
		expected := strings.TrimSpace(malicious)
		if session.Note == nil || *session.Note != expected {
			t.Fatalf("malicious input not preserved: expected %q, got %v", expected, session.Note)
		}
	})
}
