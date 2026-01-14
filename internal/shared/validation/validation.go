// Package validation provides input validation and sanitization for the time tracker.
package validation

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeString cleans a string input by:
// - Trimming leading/trailing whitespace
// - Removing null bytes (which can cause issues in databases)
// - Ensuring valid UTF-8 encoding
// The function preserves special characters like SQL injection attempts and XSS scripts
// as raw text (they are stored safely, not executed).
func SanitizeString(s string) string {
	// Remove null bytes which can cause issues
	s = strings.ReplaceAll(s, "\x00", "")

	// Ensure valid UTF-8 by replacing invalid sequences
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}

	// Trim leading/trailing whitespace
	s = strings.TrimSpace(s)

	return s
}

// SanitizeStringPtr sanitizes a string pointer, returning nil if the result is empty.
func SanitizeStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	sanitized := SanitizeString(*s)
	if sanitized == "" {
		return nil
	}
	return &sanitized
}

// ContainsControlChars checks if a string contains control characters
// (except for common whitespace like space, tab, newline).
func ContainsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// IsValidUTF8 checks if a string is valid UTF-8.
func IsValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

// RemoveControlChars removes control characters from a string,
// preserving common whitespace (space, tab, newline, carriage return).
func RemoveControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return -1 // Remove the character
		}
		return r
	}, s)
}

// SanitizeQueryParam sanitizes a query parameter string.
// This is used for filter parameters like category and search terms.
// Special characters are preserved as they are safely used in parameterized queries.
func SanitizeQueryParam(s string) string {
	return SanitizeString(s)
}

// SanitizeQueryParamPtr sanitizes a query parameter string pointer.
func SanitizeQueryParamPtr(s *string) *string {
	return SanitizeStringPtr(s)
}

// ParseIntParam parses an integer query parameter with bounds checking.
// Returns the default value if parsing fails or value is out of bounds.
func ParseIntParam(s string, defaultVal, minVal, maxVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// ValidateStringLength checks if a string length is within bounds.
func ValidateStringLength(s string, minLen, maxLen int) bool {
	length := len(s)
	return length >= minLen && length <= maxLen
}

// TruncateString truncates a string to the specified maximum length.
// If the string is already within the limit, it is returned unchanged.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
