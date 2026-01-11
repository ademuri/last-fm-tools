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

	// 4. Listens
	now := time.Now().Unix()
	longAgo := time.Now().AddDate(-2, 0, 0).Unix()

	// Current listens
	for i := 0; i < 10; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, now-int64(i*3600))
	}
	// Historical listens
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, longAgo-int64(i*3600))
	}
	// Listens for artist with no tags
	db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 3, ?)", user, now)

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