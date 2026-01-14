package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// Feature: time-tracker, Property 12: 安全头正确性
// *For any* API 响应，都应包含以下安全头：
// - X-Content-Type-Options: nosniff
// - X-Frame-Options: DENY
// - Content-Security-Policy
// **Validates: Requirements 4.9**

func TestSecurityHeaders_Property12(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random HTTP method
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
		// Generate random path
		path := "/" + rapid.StringMatching(`[a-z]{1,10}(/[a-z]{1,10})?`).Draw(t, "path")
		// Generate random status code that handler returns
		statusCode := rapid.SampledFrom([]int{200, 201, 400, 404, 500}).Draw(t, "statusCode")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
			w.Write([]byte("response"))
		})

		req := httptest.NewRequest(method, path, nil)
		rr := httptest.NewRecorder()

		SecurityHeadersMiddleware(handler).ServeHTTP(rr, req)

		// Check all required security headers
		requiredHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'",
		}

		for header, expected := range requiredHeaders {
			got := rr.Header().Get(header)
			if got != expected {
				t.Fatalf("header %s = %q, want %q", header, got, expected)
			}
		}
	})
}
