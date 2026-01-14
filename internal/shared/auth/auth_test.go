package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		expected string
		want     bool
	}{
		{"valid key", "test-api-key-32-chars-minimum!!", "test-api-key-32-chars-minimum!!", true},
		{"invalid key", "wrong-key", "test-api-key-32-chars-minimum!!", false},
		{"empty provided", "", "test-api-key-32-chars-minimum!!", false},
		{"empty expected", "test-api-key-32-chars-minimum!!", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyAPIKey(tt.provided, tt.expected); got != tt.want {
				t.Errorf("VerifyAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyBasicAuth(t *testing.T) {
	validAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123"))
	wrongPassAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrongpass"))
	wrongUserAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("wrong:secret123"))
	invalidBase64 := "Basic not-valid-base64!!!"
	noColonAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("nocolon"))

	tests := []struct {
		name         string
		authHeader   string
		expectedUser string
		expectedPass string
		want         bool
	}{
		{"valid credentials", validAuth, "admin", "secret123", true},
		{"wrong password", wrongPassAuth, "admin", "secret123", false},
		{"wrong username", wrongUserAuth, "admin", "secret123", false},
		{"empty header", "", "admin", "secret123", false},
		{"no Basic prefix", "Bearer token", "admin", "secret123", false},
		{"invalid base64", invalidBase64, "admin", "secret123", false},
		{"no colon in credentials", noColonAuth, "admin", "secret123", false},
		{"empty expected user", validAuth, "", "secret123", false},
		{"empty expected pass", validAuth, "admin", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyBasicAuth(tt.authHeader, tt.expectedUser, tt.expectedPass); got != tt.want {
				t.Errorf("VerifyBasicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	expectedKey := "test-api-key-32-chars-minimum!!"
	middleware := APIKeyMiddleware(expectedKey, "", "")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	t.Run("valid API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", expectedKey)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("missing API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("invalid API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}

func TestBasicAuthMiddleware(t *testing.T) {
	expectedUser := "admin"
	expectedPass := "secret123"
	middleware := BasicAuthMiddleware(expectedUser, expectedPass)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	validAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123"))

	t.Run("valid credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/web/test", nil)
		req.Header.Set("Authorization", validAuth)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/web/test", nil)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
		if rr.Header().Get("WWW-Authenticate") == "" {
			t.Error("expected WWW-Authenticate header")
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		wrongAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrong"))
		req := httptest.NewRequest("GET", "/web/test", nil)
		req.Header.Set("Authorization", wrongAuth)
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}
