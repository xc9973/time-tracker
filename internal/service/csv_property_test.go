package service

import (
	"bytes"
	"encoding/csv"
	"os"
	"regexp"
	"testing"

	"pgregory.net/rapid"
	"time-tracker/internal/models"
	"time-tracker/internal/repository"

	"time-tracker/internal/shared/database"
)

// Feature: time-tracker, Property 8: CSV 导出格式正确性
// **Validates: Requirements 3.1, 3.2, 3.3, 3.5**
//
// For any CSV export request:
// - Response Content-Type is text/csv
// - Content starts with UTF-8 BOM (0xEF 0xBB 0xBF)
// - Sessions CSV duration format is H:MM:SS

func setupCSVTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "service_csv_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := database.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestCSVExport_Property8_SessionsFormatCorrectness(t *testing.T) {
	db, cleanup := setupCSVTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	sessionSvc := NewSessionService(sessionRepo)

	// Duration format regex: H:MM:SS (e.g., 0:00:00, 1:23:45, 12:34:56)
	durationRegex := regexp.MustCompile(`^\d+:\d{2}:\d{2}$`)

	rapid.Check(t, func(t *rapid.T) {
		// Generate random session data
		category := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category")
		task := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task")

		// Create and stop a session
		_, err := sessionSvc.StartSession(&models.SessionStart{
			Category: category,
			Task:     task,
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}

		_, err = sessionSvc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}

		// Export CSV
		csvData, err := sessionSvc.ExportCSV(nil, nil)
		if err != nil {
			t.Fatalf("failed to export CSV: %v", err)
		}

		// Verify UTF-8 BOM
		if len(csvData) < 3 {
			t.Fatal("CSV data too short")
		}
		if csvData[0] != 0xEF || csvData[1] != 0xBB || csvData[2] != 0xBF {
			t.Fatal("CSV does not start with UTF-8 BOM")
		}

		// Verify CSV is parseable
		reader := csv.NewReader(bytes.NewReader(csvData[3:])) // Skip BOM
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Verify header row exists
		if len(records) < 1 {
			t.Fatal("CSV has no header row")
		}

		expectedHeader := []string{"id", "category", "task", "note", "location", "mood", "started_at", "ended_at", "duration", "status"}
		if len(records[0]) != len(expectedHeader) {
			t.Fatalf("expected %d columns, got %d", len(expectedHeader), len(records[0]))
		}
		for i, col := range expectedHeader {
			if records[0][i] != col {
				t.Fatalf("expected column %d to be %q, got %q", i, col, records[0][i])
			}
		}

		// Verify duration format for stopped sessions (H:MM:SS)
		durationColIdx := 8 // duration column index
		for i := 1; i < len(records); i++ {
			duration := records[i][durationColIdx]
			status := records[i][9] // status column

			// Only stopped sessions should have duration
			if status == "stopped" && duration != "" {
				if !durationRegex.MatchString(duration) {
					t.Fatalf("invalid duration format %q, expected H:MM:SS", duration)
				}
			}
		}
	})
}


// Feature: time-tracker, Property 9: CSV 导出过滤一致性
// **Validates: Requirements 3.4**
//
// For the same filter conditions, CSV export record count and content
// should match the list API results.

func TestCSVExport_Property9_SessionsFilterConsistency(t *testing.T) {
	db, cleanup := setupCSVTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	sessionSvc := NewSessionService(sessionRepo)

	// Create test data with different categories and statuses
	categories := []string{"work", "personal", "study"}
	for i := 0; i < 9; i++ {
		cat := categories[i%len(categories)]
		_, err := sessionSvc.StartSession(&models.SessionStart{
			Category: cat,
			Task:     "task_" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}

		// Stop some sessions (leave some running would cause conflict, so stop all)
		_, err = sessionSvc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pick random filters
		var status *string
		if rapid.Bool().Draw(t, "hasStatus") {
			s := rapid.SampledFrom([]string{"running", "stopped"}).Draw(t, "status")
			status = &s
		}

		var category *string
		if rapid.Bool().Draw(t, "hasCategory") {
			cat := rapid.SampledFrom(categories).Draw(t, "category")
			category = &cat
		}

		// Get list results
		listResult, err := sessionSvc.GetSessions(10000, 0, status, category)
		if err != nil {
			t.Fatalf("failed to get sessions: %v", err)
		}

		// Get CSV export
		csvData, err := sessionSvc.ExportCSV(status, category)
		if err != nil {
			t.Fatalf("failed to export CSV: %v", err)
		}

		// Parse CSV
		reader := csv.NewReader(bytes.NewReader(csvData[3:])) // Skip BOM
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// CSV has header row, so data rows = len(records) - 1
		csvDataRows := len(records) - 1
		if csvDataRows < 0 {
			csvDataRows = 0
		}

		// Verify count matches
		if csvDataRows != len(listResult.Items) {
			t.Fatalf("CSV has %d data rows, but list returned %d items", csvDataRows, len(listResult.Items))
		}

		// Verify content matches (check filters)
		for i := 1; i < len(records); i++ {
			csvCategory := records[i][1] // category is column 1
			csvStatus := records[i][9]   // status is column 9

			if category != nil && csvCategory != *category {
				t.Fatalf("CSV row %d has category %q, expected %q", i, csvCategory, *category)
			}
			if status != nil && csvStatus != *status {
				t.Fatalf("CSV row %d has status %q, expected %q", i, csvStatus, *status)
			}
		}
	})
}

