package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// Feature: time-tracker, Property 10: API Key 认证正确性
// *For any* API 请求到 /api/* 端点：
// - 无 X-API-Key 头时返回 401
// - X-API-Key 值与配置不匹配时返回 401
// - X-API-Key 值正确时正常处理请求
// **Validates: Requirements 4.1, 4.2, 4.3**

func TestAPIKeyAuth_Property10_MissingKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid API key (at least 32 chars)
		expectedKey := rapid.StringMatching(`[a-zA-Z0-9]{32,64}`).Draw(t, "expectedKey")
		middleware := APIKeyMiddleware(expectedKey, "", "")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Request without X-API-Key header
		req := httptest.NewRequest("GET", "/api/test", nil)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		// Should return 401 Unauthorized
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("missing API key should return 401, got %d", rr.Code)
		}
	})
}

func TestAPIKeyAuth_Property10_InvalidKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate expected and provided keys that are different
		expectedKey := rapid.StringMatching(`[a-zA-Z0-9]{32,64}`).Draw(t, "expectedKey")
		providedKey := rapid.StringMatching(`[a-zA-Z0-9]{1,64}`).Draw(t, "providedKey")

		// Skip if keys happen to match
		if providedKey == expectedKey {
			return
		}

		middleware := APIKeyMiddleware(expectedKey, "", "")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", providedKey)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		// Should return 401 Unauthorized
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("invalid API key should return 401, got %d", rr.Code)
		}
	})
}

func TestAPIKeyAuth_Property10_ValidKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid API key
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{32,64}`).Draw(t, "apiKey")
		middleware := APIKeyMiddleware(apiKey, "", "")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		// Should return 200 OK (request processed normally)
		if rr.Code != http.StatusOK {
			t.Fatalf("valid API key should return 200, got %d", rr.Code)
		}
	})
}

// Feature: time-tracker, Property 10: Basic Auth 认证正确性 (part of Property 10)
// **Validates: Requirements 4.11**

func TestBasicAuth_Property10_ValidCredentials(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random username and password
		user := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(t, "user")
		pass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "pass")

		middleware := BasicAuthMiddleware(user, pass)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Create valid Basic Auth header
		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
		req := httptest.NewRequest("GET", "/web/test", nil)
		req.Header.Set("Authorization", authHeader)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("valid credentials should return 200, got %d", rr.Code)
		}
	})
}

func TestBasicAuth_Property10_InvalidCredentials(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate expected and provided credentials
		expectedUser := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(t, "expectedUser")
		expectedPass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "expectedPass")
		providedUser := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(t, "providedUser")
		providedPass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "providedPass")

		// Skip if credentials happen to match
		if providedUser == expectedUser && providedPass == expectedPass {
			return
		}

		middleware := BasicAuthMiddleware(expectedUser, expectedPass)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(providedUser+":"+providedPass))
		req := httptest.NewRequest("GET", "/web/test", nil)
		req.Header.Set("Authorization", authHeader)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("invalid credentials should return 401, got %d", rr.Code)
		}
	})
}
