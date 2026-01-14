package middleware

import (
	"net/http"
)

// SecurityHeaders defines the security headers to be added to responses.
var SecurityHeaders = map[string]string{
	"X-Content-Type-Options": "nosniff",
	"X-Frame-Options":        "DENY",
	"X-XSS-Protection":       "1; mode=block",
}

type CSPNonceKey struct{}

// SecurityHeadersMiddleware adds security headers to all responses.
// Headers added:
// - X-Content-Type-Options: nosniff
// - X-Frame-Options: DENY
// - Content-Security-Policy: default-src 'self'
// - X-XSS-Protection: 1; mode=block
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range SecurityHeaders {
			w.Header().Set(key, value)
		}

		cspValue := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'"
		if nonce, ok := r.Context().Value(CSPNonceKey{}).(string); ok {
			cspValue = "default-src 'self'; script-src 'self' 'nonce-" + nonce + "' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'"
		}
		w.Header().Set("Content-Security-Policy", cspValue)

		next.ServeHTTP(w, r)
	})
}
