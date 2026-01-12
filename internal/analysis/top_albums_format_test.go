package analysis

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
	_ "github.com/mattn/go-sqlite3"
)

func TestTopAlbumsFormat(t *testing.T) {
	// Setup DB
	dbPath := filepath.Join(t.TempDir(), "test_format.db")
	db, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer db.Close()

	user := "testuser"
	artist := "Test Artist"
	album := "Test Album"
	
	// Insert Data
	db.CreateUser(user)
	
	now := time.Now()
	// 5 listens
	var tracks []store.TrackImport
	for i := 0; i < 5; i++ {
		tracks = append(tracks, store.TrackImport{
			Artist:    artist,
			Album:     album,
			TrackName: "Track 1",
			DateUTS:   fmt.Sprintf("%d", now.Unix()-int64(i)),
		})
	}
	db.AddRecentTracks(user, tracks)

	// Test GetTopAlbumsForArtist (Direct store call)
	albumsData, err := db.GetTopAlbumsForArtist(user, artist, now.AddDate(0, -1, 0), now.AddDate(0, 1, 0), 10)
	if err != nil {
		t.Fatalf("GetTopAlbumsForArtist failed: %v", err)
	}

	if len(albumsData) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albumsData))
	}

	if albumsData[0].Tag != album { // Tag field used for Album Name in TagCount reuse
		t.Errorf("expected '%s', got '%s'", album, albumsData[0].Tag)
	}
	if albumsData[0].Count != 5 {
		t.Errorf("expected 5 listens, got %d", albumsData[0].Count)
	}
	
	// Integration Test via GenerateReport
	report, err := GenerateReport(db, user)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}
	
	expectedFormatted := "Test Album (5)"
	
	found := false
	for _, a := range report.CurrentTaste.TopArtists {
		if a.Name == artist {
			found = true
			if len(a.TopAlbums) == 0 {
				t.Errorf("artist %s has no top albums in report", artist)
			} else {
				if a.TopAlbums[0] != expectedFormatted {
					t.Errorf("in report: expected '%s', got '%s'", expectedFormatted, a.TopAlbums[0])
				}
			}
		}
	}
	if !found {
		t.Errorf("artist %s not found in report", artist)
	}
}