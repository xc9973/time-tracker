// Package auth provides authentication utilities for the time tracker.
package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// VerifyAPIKey performs constant-time comparison of API keys to prevent timing attacks.
// Returns true if the provided key matches the expected key.
func VerifyAPIKey(provided, expected string) bool {
	if len(provided) == 0 || len(expected) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// VerifyBasicAuth validates Basic Auth credentials.
// Returns true if the provided credentials match the expected username and password.
func VerifyBasicAuth(authHeader, expectedUser, expectedPass string) bool {
	if authHeader == "" || expectedUser == "" || expectedPass == "" {
		return false
	}

	// Parse "Basic <base64-encoded-credentials>"
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}

	encoded := strings.TrimPrefix(authHeader, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	// Split into username:password
	credentials := string(decoded)
	colonIdx := strings.Index(credentials, ":")
	if colonIdx < 0 {
		return false
	}

	providedUser := credentials[:colonIdx]
	providedPass := credentials[colonIdx+1:]

	// Use constant-time comparison for both username and password
	userMatch := subtle.ConstantTimeCompare([]byte(providedUser), []byte(expectedUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(providedPass), []byte(expectedPass)) == 1

	return userMatch && passMatch
}

// APIKeyMiddleware creates an HTTP middleware that validates X-API-Key header.
// It also allows Basic Auth if configured, to support web interface calls to API.
func APIKeyMiddleware(expectedKey string, basicUser, basicPass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First check API Key
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" && VerifyAPIKey(apiKey, expectedKey) {
				next.ServeHTTP(w, r)
				return
			}

			// If API Key is missing or invalid, check Basic Auth if configured
			if basicUser != "" && basicPass != "" {
				authHeader := r.Header.Get("Authorization")
				if VerifyBasicAuth(authHeader, basicUser, basicPass) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Neither valid, return unauthorized
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Invalid or missing API key"}}`))
		})
	}
}

// BasicAuthMiddleware creates an HTTP middleware that validates Basic Auth credentials.
// Returns 401 Unauthorized with WWW-Authenticate header if credentials are missing or invalid.
func BasicAuthMiddleware(expectedUser, expectedPass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !VerifyBasicAuth(authHeader, expectedUser, expectedPass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Time Tracker"`)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Invalid or missing credentials"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
