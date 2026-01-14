package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// Feature: time-tracker, Property 11: 速率限制正确性
// *For any* IP 地址，在 1 分钟内超过配置的请求限制后：
// - 返回 429 Too Many Requests
// - 响应包含 Retry-After 头
// **Validates: Requirements 4.7, 4.8**

func TestRateLimit_Property11_ExceedLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random limit between 1 and 10
		limit := rapid.IntRange(1, 10).Draw(t, "limit")
		// Generate a random IP address
		ip := rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "ip")

		limiter := NewRateLimiter(limit)
		middleware := RateLimitMiddleware(limiter)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make 'limit' requests - all should succeed
		for i := 0; i < limit; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = ip + ":12345"
			rr := httptest.NewRecorder()

			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("request %d of %d should succeed, got %d", i+1, limit, rr.Code)
			}
		}

		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = ip + ":12345"
		rr := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("request exceeding limit should return 429, got %d", rr.Code)
		}

		// Should have Retry-After header
		if rr.Header().Get("Retry-After") == "" {
			t.Fatal("rate limited response should have Retry-After header")
		}
	})
}

func TestRateLimit_Property11_DifferentIPs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a small limit
		limit := rapid.IntRange(1, 5).Draw(t, "limit")
		// Generate two different IPs
		ip1 := rapid.StringMatching(`10\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "ip1")
		ip2 := rapid.StringMatching(`192\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "ip2")

		limiter := NewRateLimiter(limit)
		middleware := RateLimitMiddleware(limiter)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Exhaust limit for ip1
		for i := 0; i < limit; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = ip1 + ":12345"
			rr := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rr, req)
		}

		// ip1 should now be rate limited
		req1 := httptest.NewRequest("GET", "/api/test", nil)
		req1.RemoteAddr = ip1 + ":12345"
		rr1 := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr1, req1)

		if rr1.Code != http.StatusTooManyRequests {
			t.Fatalf("ip1 should be rate limited, got %d", rr1.Code)
		}

		// ip2 should still be allowed
		req2 := httptest.NewRequest("GET", "/api/test", nil)
		req2.RemoteAddr = ip2 + ":12345"
		rr2 := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Fatalf("ip2 should not be rate limited, got %d", rr2.Code)
		}
	})
}
