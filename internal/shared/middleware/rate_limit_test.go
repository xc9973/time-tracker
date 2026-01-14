package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3) // 3 requests per minute

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow(ip)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	allowed, retryAfter := limiter.Allow(ip)
	if allowed {
		t.Error("4th request should be denied")
	}
	if retryAfter <= 0 {
		t.Error("retryAfter should be positive")
	}

	// Different IP should still be allowed
	allowed, _ = limiter.Allow("192.168.1.2")
	if !allowed {
		t.Error("different IP should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2) // 2 requests per minute
	middleware := RateLimitMiddleware(limiter)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, rr.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{"X-Forwarded-For single", "10.0.0.1", "", "192.168.1.1:12345", "10.0.0.1"},
		{"X-Forwarded-For multiple", "10.0.0.1, 10.0.0.2", "", "192.168.1.1:12345", "10.0.0.1"},
		{"X-Real-IP", "", "10.0.0.1", "192.168.1.1:12345", "10.0.0.1"},
		{"RemoteAddr with port", "", "", "192.168.1.1:12345", "192.168.1.1"},
		{"RemoteAddr without port", "", "", "192.168.1.1", "192.168.1.1"},
		{"X-Forwarded-For takes precedence", "10.0.0.1", "10.0.0.2", "192.168.1.1:12345", "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
