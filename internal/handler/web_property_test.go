package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"pgregory.net/rapid"

	"time-tracker/internal/sessions"
	"time-tracker/internal/shared/auth"
	"time-tracker/internal/shared/database"
)

// setupWebTestEnv creates a test environment with in-memory database.
func setupWebTestEnv(t *testing.T) (*WebHandler, func()) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "web_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	db, err := database.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create database: %v", err)
	}
	sessionRepo := sessions.NewSessionRepository(db)
	sessionSvc := sessions.NewSessionService(sessionRepo)
	// Create templates directory for testing
	tmpDir, err := os.MkdirTemp("", "templates_test")
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create temp dir: %v", err)
	}
	// Create minimal test templates
	baseHTML := `{{define "base"}}<!DOCTYPE html><html><body>{{block "content" .}}{{end}}</body></html>{{end}}`
	sessionsHTML := `{{template "base" .}}{{define "content"}}<div>Sessions: {{len .Sessions}}</div>{{end}}`
	os.WriteFile(tmpDir+"/base.html", []byte(baseHTML), 0644)
	os.WriteFile(tmpDir+"/sessions.html", []byte(sessionsHTML), 0644)

	tz, _ := time.LoadLocation("Asia/Shanghai")
	apiKey := "test-api-key-32-characters-long"
	handler, err := NewWebHandler(sessionSvc, tmpDir, tz, apiKey)
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create web handler: %v", err)
	}
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
		os.RemoveAll(tmpDir)
	}
	return handler, cleanup
}
// Feature: time-tracker, Property 15: Web Basic Auth 正确性
// *For any* 访问 /web/* 或 /*.csv 端点的请求（当配置了 Basic Auth 时）：
// - 无 Authorization 头时返回 401
// - 凭据不正确时返回 401
// - 凭据正确时正常返回页面/文件
// **Validates: Requirements 4.11, 4.12**
func TestWebBasicAuth_Property15_MissingAuth(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random credentials for the middleware
		user := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(rt, "user")
		pass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(rt, "pass")
		handler, cleanup := setupWebTestEnv(t)
		defer cleanup()
		// Wrap handler with Basic Auth middleware
		middleware := auth.BasicAuthMiddleware(user, pass)
		protectedHandler := middleware(handler)
		// Generate random path under /web/
		paths := []string{"/web/sessions"}
		pathIdx := rapid.IntRange(0, len(paths)-1).Draw(rt, "pathIdx")
		path := paths[pathIdx]
		// Request without Authorization header
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		// Should return 401 Unauthorized
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("missing auth should return 401, got %d for path %s", rr.Code, path)
		}
		// Should include WWW-Authenticate header
		if rr.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("401 response should include WWW-Authenticate header")
		}
	})
}
func TestWebBasicAuth_Property15_InvalidAuth(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate expected and provided credentials
		expectedUser := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(rt, "expectedUser")
		expectedPass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(rt, "expectedPass")
		providedUser := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(rt, "providedUser")
		providedPass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(rt, "providedPass")
		// Skip if credentials happen to match
		if providedUser == expectedUser && providedPass == expectedPass {
			return
		}
		handler, cleanup := setupWebTestEnv(t)
		defer cleanup()
		middleware := auth.BasicAuthMiddleware(expectedUser, expectedPass)
		protectedHandler := middleware(handler)
		paths := []string{"/web/sessions"}
		pathIdx := rapid.IntRange(0, len(paths)-1).Draw(rt, "pathIdx")
		path := paths[pathIdx]
		// Create invalid Basic Auth header
		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(providedUser+":"+providedPass))
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", authHeader)
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		// Should return 401 Unauthorized
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("invalid auth should return 401, got %d for path %s", rr.Code, path)
		}
	})
}
func TestWebBasicAuth_Property15_ValidAuth(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random credentials
		user := rapid.StringMatching(`[a-zA-Z0-9]{4,20}`).Draw(rt, "user")
		pass := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(rt, "pass")
		handler, cleanup := setupWebTestEnv(t)
		defer cleanup()
		middleware := auth.BasicAuthMiddleware(user, pass)
		protectedHandler := middleware(handler)
		paths := []string{"/web/sessions"}
		pathIdx := rapid.IntRange(0, len(paths)-1).Draw(rt, "pathIdx")
		path := paths[pathIdx]
		// Create valid Basic Auth header
		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", authHeader)
		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)
		// Should return 200 OK (not 401 Unauthorized)
		if rr.Code == http.StatusUnauthorized {
			t.Fatalf("valid auth should not return 401 for path %s", path)
		}
		// Should return 200 OK
		if rr.Code != http.StatusOK {
			t.Fatalf("valid auth should return 200, got %d for path %s", rr.Code, path)
		}
	})
}
// Feature: time-tracker, Property 16: 时区显示正确性
// *For any* Web 页面显示的时间戳，应按配置的 TIMELOG_TZ 时区显示，而非 UTC。
// **Validates: Requirements 5.5**
func TestTimezoneDisplay_Property16_Conversion(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random UTC timestamp components
		year := rapid.IntRange(2020, 2030).Draw(rt, "year")
		month := rapid.IntRange(1, 12).Draw(rt, "month")
		day := rapid.IntRange(1, 28).Draw(rt, "day") // Use 28 to avoid month boundary issues
		hour := rapid.IntRange(0, 23).Draw(rt, "hour")
		minute := rapid.IntRange(0, 59).Draw(rt, "minute")
		// Create UTC time
		utcTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
		rfc3339 := utcTime.Format(time.RFC3339)
		// Test with different timezones
		tzNames := []string{"Asia/Shanghai", "America/New_York", "Europe/London", "UTC"}
		tzIdx := rapid.IntRange(0, len(tzNames)-1).Draw(rt, "tzIdx")
		tzName := tzNames[tzIdx]
		tz, err := time.LoadLocation(tzName)
		if err != nil {
			t.Fatalf("failed to load timezone %s: %v", tzName, err)
		}
		// Create a minimal handler just to test formatTime
		handler := &WebHandler{timezone: tz}
		formatted := handler.formatTime(rfc3339)
		// Parse the formatted time back
		expectedTime := utcTime.In(tz)
		expectedFormatted := expectedTime.Format("2006-01-02 15:04")
		if formatted != expectedFormatted {
			t.Fatalf("timezone conversion failed: input=%s, tz=%s, expected=%s, got=%s",
				rfc3339, tzName, expectedFormatted, formatted)
		}
	})
}
func TestTimezoneDisplay_Property16_NotUTC(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random UTC timestamp that would differ in Asia/Shanghai
		// Use hours that would definitely show different date/time in Shanghai (UTC+8)
		hour := rapid.IntRange(16, 23).Draw(rt, "hour") // These hours will be next day in Shanghai
		year := rapid.IntRange(2020, 2030).Draw(rt, "year")
		month := rapid.IntRange(1, 12).Draw(rt, "month")
		day := rapid.IntRange(1, 27).Draw(rt, "day")
		utcTime := time.Date(year, time.Month(month), day, hour, 30, 0, 0, time.UTC)
		rfc3339 := utcTime.Format(time.RFC3339)
		// Use Shanghai timezone (UTC+8)
	tz, _ := time.LoadLocation("Asia/Shanghai")
		handler := &WebHandler{timezone: tz}
		formatted := handler.formatTime(rfc3339)
		// The formatted time should NOT be the same as UTC formatted time
		utcFormatted := utcTime.Format("2006-01-02 15:04")
		// For hours 16-23 UTC, Shanghai time will be different (next day or different hour)
		shanghaiTime := utcTime.In(tz)
		shanghaiFormatted := shanghaiTime.Format("2006-01-02 15:04")
		if formatted == utcFormatted && shanghaiFormatted != utcFormatted {
			t.Fatalf("time should be converted to configured timezone, not UTC: utc=%s, formatted=%s, expected=%s",
				utcFormatted, formatted, shanghaiFormatted)
		}
		if formatted != shanghaiFormatted {
			t.Fatalf("formatted time doesn't match expected Shanghai time: got=%s, expected=%s",
				formatted, shanghaiFormatted)
		}
	})
}
func TestTimezoneDisplay_Property16_NilPointer(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
	tz, _ := time.LoadLocation("Asia/Shanghai")
		handler := &WebHandler{timezone: tz}
		// Test nil pointer handling
		result := handler.formatTimePtr(nil)
		if result != "" {
			t.Fatalf("formatTimePtr(nil) should return empty string, got %s", result)
		}
	})
}