package analysis

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/migration"
	_ "github.com/mattn/go-sqlite3"
)

func TestTopAlbumsFormat(t *testing.T) {
	// Setup DB
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(migration.Create)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	user := "testuser"
	artist := "Test Artist"
	album := "Test Album"
	
	// Insert Data
	db.Exec("INSERT INTO User (name) VALUES (?)", user)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", artist)
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", artist, album)
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'Track 1', ?, ?)", artist, album)
	
	now := time.Now()
	// 5 listens
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, now.Unix()-int64(i))
	}

	// Test getTopAlbumsForArtist
	albums, err := getTopAlbumsForArtist(db, user, artist, now.AddDate(0, -1, 0), now.AddDate(0, 1, 0), 10)
	if err != nil {
		t.Fatalf("getTopAlbumsForArtist failed: %v", err)
	}

	if len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}

	expected := "Test Album (5)"
	if albums[0] != expected {
		t.Errorf("expected '%s', got '%s'", expected, albums[0])
	}
	
	// Integration Test via GenerateReport
	// We need to make sure this artist is picked up as a top artist.
	// We already have listens, so it should be.
	
	// Need at least one tag or something? No, basic report should work.
	
	report, err := GenerateReport(db, user)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}
	
	found := false
	for _, a := range report.CurrentTaste.TopArtists {
		if a.Name == artist {
			found = true
			if len(a.TopAlbums) == 0 {
				t.Errorf("artist %s has no top albums in report", artist)
			} else {
				if a.TopAlbums[0] != expected {
					t.Errorf("in report: expected '%s', got '%s'", expected, a.TopAlbums[0])
				}
			}
		}
	}
	if !found {
		t.Errorf("artist %s not found in report", artist)
	}
}
