package validation

import (
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "string with leading/trailing whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "string with null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "SQL injection attempt",
			input:    "'; DROP TABLE logs; --",
			expected: "'; DROP TABLE logs; --",
		},
		{
			name:     "XSS script",
			input:    "<script>alert('XSS')</script>",
			expected: "<script>alert('XSS')</script>",
		},
		{
			name:     "Unicode characters",
			input:    "工作 学习 😀",
			expected: "工作 学习 😀",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected *string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string becomes nil",
			input:    strPtr(""),
			expected: nil,
		},
		{
			name:     "whitespace only becomes nil",
			input:    strPtr("   "),
			expected: nil,
		},
		{
			name:     "normal string",
			input:    strPtr("hello"),
			expected: strPtr("hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeStringPtr(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("SanitizeStringPtr() = %v, want nil", *result)
				}
			} else {
				if result == nil {
					t.Errorf("SanitizeStringPtr() = nil, want %q", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("SanitizeStringPtr() = %q, want %q", *result, *tt.expected)
				}
			}
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal int
		minVal     int
		maxVal     int
		expected   int
	}{
		{
			name:       "empty string returns default",
			input:      "",
			defaultVal: 50,
			minVal:     1,
			maxVal:     200,
			expected:   50,
		},
		{
			name:       "valid number",
			input:      "100",
			defaultVal: 50,
			minVal:     1,
			maxVal:     200,
			expected:   100,
		},
		{
			name:       "below minimum",
			input:      "0",
			defaultVal: 50,
			minVal:     1,
			maxVal:     200,
			expected:   1,
		},
		{
			name:       "above maximum",
			input:      "500",
			defaultVal: 50,
			minVal:     1,
			maxVal:     200,
			expected:   200,
		},
		{
			name:       "invalid number returns default",
			input:      "abc",
			defaultVal: 50,
			minVal:     1,
			maxVal:     200,
			expected:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIntParam(tt.input, tt.defaultVal, tt.minVal, tt.maxVal)
			if result != tt.expected {
				t.Errorf("ParseIntParam(%q, %d, %d, %d) = %d, want %d",
					tt.input, tt.defaultVal, tt.minVal, tt.maxVal, result, tt.expected)
			}
		})
	}
}

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minLen   int
		maxLen   int
		expected bool
	}{
		{
			name:     "valid length",
			input:    "hello",
			minLen:   1,
			maxLen:   10,
			expected: true,
		},
		{
			name:     "too short",
			input:    "",
			minLen:   1,
			maxLen:   10,
			expected: false,
		},
		{
			name:     "too long",
			input:    "hello world!",
			minLen:   1,
			maxLen:   10,
			expected: false,
		},
		{
			name:     "exact minimum",
			input:    "a",
			minLen:   1,
			maxLen:   10,
			expected: true,
		},
		{
			name:     "exact maximum",
			input:    "1234567890",
			minLen:   1,
			maxLen:   10,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateStringLength(tt.input, tt.minLen, tt.maxLen)
			if result != tt.expected {
				t.Errorf("ValidateStringLength(%q, %d, %d) = %v, want %v",
					tt.input, tt.minLen, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "truncation needed",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestContainsControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "normal string",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "string with tab",
			input:    "hello\tworld",
			expected: false,
		},
		{
			name:     "string with newline",
			input:    "hello\nworld",
			expected: false,
		},
		{
			name:     "string with null byte",
			input:    "hello\x00world",
			expected: true,
		},
		{
			name:     "string with bell character",
			input:    "hello\x07world",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsControlChars(tt.input)
			if result != tt.expected {
				t.Errorf("ContainsControlChars(%q) = %v, want %v",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "preserves tab",
			input:    "hello\tworld",
			expected: "hello\tworld",
		},
		{
			name:     "preserves newline",
			input:    "hello\nworld",
			expected: "hello\nworld",
		},
		{
			name:     "removes null byte",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "removes bell character",
			input:    "hello\x07world",
			expected: "helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveControlChars(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveControlChars(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
