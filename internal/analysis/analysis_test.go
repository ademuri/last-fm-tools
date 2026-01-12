package analysis

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/migration"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	_, err = db.Exec(migration.Create)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	return db
}

func TestGenerateReport(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	// Insert dummy data
	// 1. User
	db.Exec("INSERT INTO User (name) VALUES (?)", user)

	// 2. Artists
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist A")
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist B")
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist C") // No tags
	
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist A", "Album A1")
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist B", "Album B1")
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist C", "Album C1") // No tags

	// 3. Tracks
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'Track 1', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (2, 'Track 2', 'Artist B', 'Album B1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (3, 'Track 3', 'Artist C', 'Album C1')")
	// Add more tracks to Album A1 to trigger "album-oriented" (Avg tracks > 3)
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (4, 'Track 4', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (5, 'Track 5', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (6, 'Track 6', 'Artist A', 'Album A1')")

	// 4. Listens
	now := time.Now().Unix()
	longAgo := time.Now().AddDate(-2, 0, 0).Unix()

	// Current listens
	// Listen to Album A1 (4 tracks)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, now-100)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 4, ?)", user, now-200)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 5, ?)", user, now-300)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 6, ?)", user, now-400)
	// Listen to Album B1 (1 track)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, now-500)
	// Listen to Album C1 (1 track)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 3, ?)", user, now-600)

	// Avg tracks/album: (4 + 1 + 1) / 3 = 2.0. Still track oriented? 
	// Wait, threshold is 3.0. 
	// Let's add more tracks to A1.
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (7, 'Track 7', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (8, 'Track 8', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 7, ?)", user, now-700)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 8, ?)", user, now-800)
	
	// Avg: (6 + 1 + 1) / 3 = 2.66. Still under 3.0.
	// Let's add Album A2 with 5 tracks.
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist A", "Album A2")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (11, 'T1', 'Artist A', 'Album A2')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (12, 'T2', 'Artist A', 'Album A2')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (13, 'T3', 'Artist A', 'Album A2')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (14, 'T4', 'Artist A', 'Album A2')")
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 11, ?)", user, now-1000)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 12, ?)", user, now-1100)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 13, ?)", user, now-1200)
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 14, ?)", user, now-1300)

	// Albums: A1 (6 tracks), B1 (1 track), C1 (1 track), A2 (4 tracks)
	// Avg: (6+1+1+4)/4 = 3.0 -> Album Oriented.
	
	// Historical listens
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, longAgo-int64(i*3600))
	}

	// 5. Tags
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Rock")
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Indie")
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Alternative")
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Pop") 
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Cool") 
	
	// Artist A: Rock (100), Indie (80) -> Valid
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist A", "Rock", 100)
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist A", "Indie", 80)
	
	// Artist B: Rock (50), Alternative (30) -> Valid
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist B", "Rock", 50)
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist B", "Alternative", 30)

	// Album Tags for Album A1 -> Valid
	db.Exec("INSERT INTO AlbumTag (artist, album, tag, count) VALUES (?, ?, ?, ?)", "Artist A", "Album A1", "Pop", 60)
	db.Exec("INSERT INTO AlbumTag (artist, album, tag, count) VALUES (?, ?, ?, ?)", "Artist A", "Album A1", "Cool", 40)

	report, err := GenerateReport(db, user)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report.Metadata.TotalScrobbles != 17 { // 12 current + 5 historical
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

	// Verify Album C1 (no tags) is handled
	foundC1 := false
	for _, a := range report.CurrentTaste.TopAlbums {
		if a.Title == "Album C1" {
			foundC1 = true
			if len(a.Tags) != 0 {
				t.Errorf("expected Album C1 to have 0 tags, got %d", len(a.Tags))
			}
		}
	}
	if !foundC1 {
		t.Errorf("Album C1 not found in top albums")
	}
	// Verify Listening Patterns
	// Artists: 
	// A: 2 albums (A1, A2). Wait, A1 has tracks, A2 has tracks.
	// B: 1 album (B1).
	// C: 1 album (C1).
	// Total 3 artists.
	// Albums per artist: A=2, B=1, C=1.
	// Median: 1, 1, 2 -> 1.
	// Average: (2+1+1)/3 = 1.33 -> 1.3
	
	if report.ListeningPatterns.AllAlbumsPerArtistMedian != 1.0 {
		t.Errorf("expected median 1.0, got %f", report.ListeningPatterns.AllAlbumsPerArtistMedian)
	}
	if report.ListeningPatterns.AllAlbumsPerArtistAverage != 1.3 {
		t.Errorf("expected average 1.3, got %f", report.ListeningPatterns.AllAlbumsPerArtistAverage)
	}
	// Top 100 is same as All since only 3 artists
	if report.ListeningPatterns.Top100ArtistsAlbumsMedian != 1.0 {
		t.Errorf("expected top 100 median 1.0, got %f", report.ListeningPatterns.Top100ArtistsAlbumsMedian)
	}
	// Artists with 3+ albums: 0
	if report.ListeningPatterns.ArtistsWith3PlusAlbums != 0 {
		t.Errorf("expected 0 artists with 3+ albums, got %d", report.ListeningPatterns.ArtistsWith3PlusAlbums)
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

func TestGetPeakYears(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	artist := "Artist A"
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'T', ?, 'A')", artist)

	years := []int{2020, 2021, 2022}
	for _, y := range years {
		ts := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		for i := 0; i < 10; i++ {
			db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, ts)
		}
	}

	peak, err := getPeakYears(db, user, artist)
	if err != nil {
		t.Fatalf("getPeakYears failed: %v", err)
	}

	expected := "2020-2022"
	if peak != expected {
		t.Errorf("expected %s, got %s", expected, peak)
	}
}