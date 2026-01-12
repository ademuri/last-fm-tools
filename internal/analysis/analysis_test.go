package analysis

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *store.Store {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func TestGenerateReport(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user)

	now := time.Now()
	longAgo := now.AddDate(-2, 0, 0)

	// Helper to insert listen
	addListen := func(artist, album, track string, ts time.Time) {
		tracks := []store.TrackImport{
			{
				Artist:    artist,
				Album:     album,
				TrackName: track,
				DateUTS:   fmt.Sprintf("%d", ts.Unix()),
			},
		}
		if err := db.AddRecentTracks(user, tracks); err != nil {
			t.Fatalf("failed to add listen: %v", err)
		}
	}

	// Current listens (Last 18 months)
	// Album A1 (Artist A)
	addListen("Artist A", "Album A1", "Track 1", now.Add(-100*time.Second))
	addListen("Artist A", "Album A1", "Track 4", now.Add(-200*time.Second))
	addListen("Artist A", "Album A1", "Track 5", now.Add(-300*time.Second))
	addListen("Artist A", "Album A1", "Track 6", now.Add(-400*time.Second))
	
	// Album B1 (Artist B)
	addListen("Artist B", "Album B1", "Track 2", now.Add(-500*time.Second))
	
	// Album C1 (Artist C)
	addListen("Artist C", "Album C1", "Track 3", now.Add(-600*time.Second))

	// Add more tracks to A1 to trigger "album-oriented" (Avg tracks >= 3)
	// Currently A1: 4 tracks. B1: 1. C1: 1. Avg = (4+1+1)/3 = 2.0.
	// Need more.
	addListen("Artist A", "Album A1", "Track 7", now.Add(-700*time.Second))
	addListen("Artist A", "Album A1", "Track 8", now.Add(-800*time.Second))
	// A1 now 6 tracks. Avg = (6+1+1)/3 = 2.66.

	// Add Album A2 (Artist A) with 4 tracks
	addListen("Artist A", "Album A2", "T1", now.Add(-1000*time.Second))
	addListen("Artist A", "Album A2", "T2", now.Add(-1100*time.Second))
	addListen("Artist A", "Album A2", "T3", now.Add(-1200*time.Second))
	addListen("Artist A", "Album A2", "T4", now.Add(-1300*time.Second))
	
	// Albums: A1 (6), B1 (1), C1 (1), A2 (4). Avg = (6+1+1+4)/4 = 3.0.

	// Historical listens
	for i := 0; i < 5; i++ {
		addListen("Artist B", "Album B1", "Track 2", longAgo.Add(time.Duration(i)*time.Hour))
	}

	// Tags
	db.SaveArtistTags("Artist A", []string{"Rock", "Indie"}, []int{100, 80})
	db.SaveArtistTags("Artist B", []string{"Rock", "Alternative"}, []int{50, 30})
	
	db.SaveAlbumTags("Artist A", "Album A1", []string{"Pop", "Cool"}, []int{60, 40})

	report, err := GenerateReport(db, user)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	// 12 current + 5 historical = 17
	if report.Metadata.TotalScrobbles != 17 { 
		t.Errorf("expected 17 scrobbles, got %d", report.Metadata.TotalScrobbles)
	}

	if report.Metadata.ListeningStyle != "album-oriented" {
		t.Errorf("expected album-oriented style, got %s", report.Metadata.ListeningStyle)
	}

	if len(report.CurrentTaste.TopArtists) == 0 {
		t.Errorf("expected at least one current artist")
	} else if report.CurrentTaste.TopArtists[0].Name != "Artist A" {
		t.Errorf("expected Artist A to be top current artist, got %s", report.CurrentTaste.TopArtists[0].Name)
	}

	// Verify Artist C (no tags) is handled
	foundC := false
	for _, a := range report.CurrentTaste.TopArtists {
		if a.Name == "Artist C" {
			foundC = true
			if len(a.PrimaryTags) != 0 {
				t.Errorf("expected Artist C to have 0 tags, got %d", len(a.PrimaryTags))
			}
		}
	}
	if !foundC {
		t.Errorf("Artist C not found in top artists")
	}

	// Verify Listening Patterns
	if report.ListeningPatterns.AllAlbumsPerArtistMedian != 1.0 {
		t.Errorf("expected median 1.0, got %f", report.ListeningPatterns.AllAlbumsPerArtistMedian)
	}
	if report.ListeningPatterns.AllAlbumsPerArtistAverage != 1.3 {
		t.Errorf("expected average 1.3, got %f", report.ListeningPatterns.AllAlbumsPerArtistAverage)
	}
}

func TestFilterTags(t *testing.T) {
	tags := []string{"Valid1", "Valid2", "1999", "Tiny", "LowWeight", "With-Hyphen", "With_Underscore"}
	counts := []int{100, 50, 100, 100, 10, 100, 100}
	
	filtered := filterTags(tags, counts)
	
	expected := []string{"valid1", "valid2", "tiny", "with hyphen", "with underscore"}
	
	if len(filtered) != len(expected) {
		t.Errorf("expected %d tags, got %d: %v", len(expected), len(filtered), filtered)
	}
	
	for i, ex := range expected {
		if i < len(filtered) && filtered[i] != ex {
			t.Errorf("expected tag %s, got %s", ex, filtered[i])
		}
	}
}

func TestCalculateDrift(t *testing.T) {
	hist := []TagStat{
		{Tag: "rock", Weight: 0.8},
		{Tag: "jazz", Weight: 0.5}, 
	}
	curr := []TagStat{
		{Tag: "rock", Weight: 0.9},
		{Tag: "techno", Weight: 0.6}, 
	}
	
	declined, emerged := calculateDrift(hist, curr)
	
	if len(declined) != 1 || declined[0].Tag != "jazz" {
		t.Errorf("expected 'jazz' to decline")
	}
	
	if len(emerged) != 1 || emerged[0].Tag != "techno" {
		t.Errorf("expected 'techno' to emerge")
	}
}
