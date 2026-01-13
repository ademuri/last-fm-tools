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
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()
		
		// Populate with recent data
		now := time.Now()
		for i := 0; i < 10; i++ {
			d := now.AddDate(0, 0, -i)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.Local))
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{}
		analyzer.Configure(map[string]string{"days": "14"}) // Set defaults + days
		
		_, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != ErrSkipReport {
			t.Errorf("Expected ErrSkipReport, got %v", err)
		}
	})

	t.Run("Work Failure", func(t *testing.T) {
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()

		now := time.Now()
		for i := 0; i < 10; i++ {
			d := now.AddDate(0, 0, -i)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{}
		analyzer.Configure(map[string]string{"days": "14"})

		res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
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
		for i := 0; i < 20; i++ {
			d := now.AddDate(0, 0, -i)
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
				continue // Skip weekend
			}
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.Local))
		}

		analyzer := &CheckSourcesAnalyzer{}
		analyzer.Configure(map[string]string{"days": "20"})

		res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "Potential Weekend Scrobbler Failure"
		if !contains(res.BodyOverride, expected) {
			t.Errorf("Expected report to contain %q, got: %s", expected, res.BodyOverride)
		}
	})

	t.Run("Sensitivity Configuration", func(t *testing.T) {
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()

		now := time.Now()
		// Create a scenario: 5 days of silence during Work Hours
		// Add listens for 10 days, but SKIP work hours for the last 5 days
		for i := 0; i < 10; i++ {
			d := now.AddDate(0, 0, -i)
			// Always add Other Hours
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
			
			// Add Work Hours ONLY for days 6-10 (older)
			if i >= 6 {
				addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.Local))
			}
		}
		// Result: Streak should be 5 working days (assuming M-F).
		// Wait, if today is Wed, last 5 days are Wed, Tue, Mon, Sun, Sat.
		// Sun/Sat skipped.
		// Streak = Wed, Tue, Mon. Streak = 3.
		// Need to ensure enough working days are silent.
		// Let's create silence for the last 10 days to be safe (streak will be ~7-8 working days).
		
		// Re-do: Silence for last 10 days
		// Actually, let's just reuse the "Work Failure" setup which had 10 days silence.
		// Streak will be ~8 days.
		
		// Test A: Default Threshold (3). Should Alert.
		analyzerDefault := &CheckSourcesAnalyzer{}
		analyzerDefault.Configure(map[string]string{"days": "14"})
		res, err := analyzerDefault.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != nil {
			t.Errorf("Default: Unexpected error: %v", err)
		}
		if !contains(res.BodyOverride, "Potential Work Scrobbler Failure") {
			t.Errorf("Default: Expected alert, got none")
		}

		// Test B: High Threshold (10). Streak (8) < 10. Should NOT Alert.
		analyzerHigh := &CheckSourcesAnalyzer{}
		analyzerHigh.Configure(map[string]string{"days": "14", "work_streak": "10"})
		_, err = analyzerHigh.GetResults(dbPath, user, time.Time{}, time.Time{})
		if err != ErrSkipReport {
			t.Errorf("High Threshold: Expected ErrSkipReport, got %v", err)
		}
	})
	t.Run("Work Hours Configuration", func(t *testing.T) {
		// Custom Window: 10:00 - 18:00.
		// Default is 9-17.
		
		// Subtest A: Listen at 09:30 (Other Hours). Window (10-18) is silent. Expect Alert.
		t.Run("Alert Triggered", func(t *testing.T) {
			dbRaw, dbPath := createTestDb(t)
			defer dbRaw.Close()
			
			for i := 0; i < 10; i++ {
				d := time.Now().AddDate(0, 0, -i)
				// Listen at 09:30.
				// If default (9-17), this is WORK. Streak = 0. No Alert.
				// If custom (10-18), this is OTHER. Work Streak = 10. Alert.
				addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 9, 30, 0, 0, time.Local))
			}

			analyzer := &CheckSourcesAnalyzer{}
			analyzer.Configure(map[string]string{"days": "14", "work_hours": "10-18"})
			
			res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !contains(res.BodyOverride, "Potential Work Scrobbler Failure") {
				t.Errorf("Expected alert with custom hours, got none")
			}
		})

		// Subtest B: Listen at 10:30 (Work Hours). Expect No Alert.
		t.Run("Alert Suppressed", func(t *testing.T) {
			dbRaw, dbPath := createTestDb(t)
			defer dbRaw.Close()
			
			for i := 0; i < 10; i++ {
				d := time.Now().AddDate(0, 0, -i)
				addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 10, 30, 0, 0, time.Local))
			}

			analyzer := &CheckSourcesAnalyzer{}
			analyzer.Configure(map[string]string{"days": "14", "work_hours": "10-18"})
			
			_, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
			if err != ErrSkipReport {
				t.Errorf("Expected ErrSkipReport, got %v", err)
			}
		})
	})

	t.Run("Cooldown Logic", func(t *testing.T) {
		dbRaw, dbPath := createTestDb(t)
		defer dbRaw.Close()

		// Setup failure scenario (10 days silence)
		for i := 0; i < 10; i++ {
			d := time.Now().AddDate(0, 0, -i)
			addListenForDB(t, dbPath, user, time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, time.Local))
		}

		// 1. Run with Cooldown = 1. Should Warn + Update DB.
		oldCooldown := cooldownDays
		cooldownDays = 1
		defer func() { cooldownDays = oldCooldown }()

		err := checkSources(dbPath, user, 14, 0)
		if err != nil {
			t.Errorf("checkSources failed: %v", err)
		}

		// Check DB for CommandHistory
		s, _ := store.New(dbPath)
		lastRun, err := s.GetCommandLastRun(user, "check-sources")
		s.Close()
		if err != nil {
			t.Fatalf("GetCommandLastRun failed: %v", err)
		}
		if lastRun.IsZero() {
			t.Error("Expected lastRun to be set, got Zero")
		}

		// 2. Run again immediately. Should Suppress (no DB update, but existing timestamp remains).
		// Set lastRun to 2 hours ago.
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		// Truncate to second precision because DB might lose precision or formatting might differ slightly
		twoHoursAgo = twoHoursAgo.Truncate(time.Second) 
		
		s, _ = store.New(dbPath)
		s.SetCommandLastRun(user, "check-sources", twoHoursAgo)
		s.Close()

		err = checkSources(dbPath, user, 14, 0)
		if err != nil {
			t.Errorf("checkSources failed: %v", err)
		}

		s, _ = store.New(dbPath)
		lastRunAfter, _ := s.GetCommandLastRun(user, "check-sources")
		s.Close()

		// If suppressed, lastRunAfter should still be twoHoursAgo.
		if !lastRunAfter.Equal(twoHoursAgo) {
			t.Errorf("Expected timestamp to remain unchanged (suppressed), but it changed from %v to %v", twoHoursAgo, lastRunAfter)
		}

		// 3. Run with old timestamp (> cooldown). Should Warn + Update.
		twoDaysAgo := time.Now().Add(-48 * time.Hour)
		twoDaysAgo = twoDaysAgo.Truncate(time.Second)
		s, _ = store.New(dbPath)
		s.SetCommandLastRun(user, "check-sources", twoDaysAgo)
		s.Close()

		err = checkSources(dbPath, user, 14, 0)
		if err != nil {
			t.Errorf("checkSources failed: %v", err)
		}

		s, _ = store.New(dbPath)
		lastRunAfter2, _ := s.GetCommandLastRun(user, "check-sources")
		s.Close()

		if lastRunAfter2.Equal(twoDaysAgo) {
			t.Errorf("Expected timestamp to update (warned), but it stayed %v", twoDaysAgo)
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
