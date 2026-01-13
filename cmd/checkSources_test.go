package cmd

import (
	"fmt"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
)

func TestCheckSourcesAnalyzer_GetResults(t *testing.T) {
	// Setup DB
	dbRaw, dbPath := createTestDb(t)
	defer dbRaw.Close()
	
	// We need to initialize the Store to create tables properly if createTestDb doesn't do it fully?
	// createTestDb calls createDatabase(dbPath), which is likely cmd/db_legacy.go's version.
	// Let's rely on internal/store to ensure schema is correct for GetListensInRange.
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	s.Close() // Close so we can re-open in GetResults or AddRecentTracks

	user := "testuser"

	t.Run("All Good", func(t *testing.T) {
		// Clear DB? Easier to use new DB per subtest, but createTestDb is per test function usually?
		// Actually, let's just make a new DB for each case or manage data carefully.
		// Since createTestDb uses t.TempDir(), we can call it inside subtests if we want isolated DBs.
		
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()
		
		// Populate with recent data
		now := time.Now()
		// Add listens for every day in last 10 days, inside and outside work hours
		for i := 0; i < 10; i++ {
			d := now.AddDate(0, 0, -i)
			// Work hour (12:00)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.Local))
			// Other hour (20:00)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{Days: 14}
		_, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != ErrSkipReport {
			t.Errorf("Expected ErrSkipReport, got %v", err)
		}
	})

	t.Run("Work Failure", func(t *testing.T) {
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()

		now := time.Now()
		// Add listens for last 10 days ONLY in Other Hours
		for i := 0; i < 10; i++ {
			d := now.AddDate(0, 0, -i)
			// Other hour (20:00)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{Days: 14}
		res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if res.BodyOverride == "" {
			t.Error("Expected report body, got empty")
		}
		expected := "Potential Work Scrobbler Failure"
		if !contains(res.BodyOverride, expected) {
			t.Errorf("Expected report to contain %q, got: %s", expected, res.BodyOverride)
		}
	})

	t.Run("Weekend Failure", func(t *testing.T) {
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()

		now := time.Now()
		// Add listens for Weekdays (Work and Other), but SKIP Weekends completely
		for i := 0; i < 20; i++ {
			d := now.AddDate(0, 0, -i)
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
				continue // Skip weekend
			}
			// Add Weekday listens
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{Days: 20}
		res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "Potential Weekend Scrobbler Failure"
		if !contains(res.BodyOverride, expected) {
			t.Errorf("Expected report to contain %q, got: %s", expected, res.BodyOverride)
		}
	})
}

func addListenForDB(t *testing.T, dbPath, user string, ts time.Time) {
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	s.CreateUser(user)
	err = s.AddRecentTracks(user, []store.TrackImport{
		{
			Artist:    "Artist",
			Album:     "Album",
			TrackName: "Track",
			DateUTS:   fmt.Sprintf("%d", ts.Unix()),
		},
	})
	if err != nil {
		t.Fatalf("AddRecentTracks: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && search(s, substr)
}

func search(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
