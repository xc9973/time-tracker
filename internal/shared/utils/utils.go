package utils

import (
	"fmt"
	"net/url"
	"strconv"
)

// FormatDuration formats duration in seconds to H:MM:SS format.
func FormatDuration(durationSec *int64) string {
	if durationSec == nil {
		return ""
	}
	d := *durationSec
	hours := d / 3600
	minutes := (d % 3600) / 60
	seconds := d % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// PtrToString converts a string pointer to a string, returning empty string if nil.
func PtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ParsePaginationParams parses limit and offset from query parameters.
func ParsePaginationParams(query url.Values, defaultLimit int, maxLimit int) (int, int) {
	limit := defaultLimit
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
