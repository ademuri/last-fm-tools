package store

import (
	"path/filepath"
	"testing"
	"time"
)

func createTestDb(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "lastfm.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%s) error: %v", dbPath, err)
	}

	return store
}

func TestCreateUser(t *testing.T) {
	s := createTestDb(t)
	defer s.Close()

	user := "testuser"
	err := s.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser(%q) error: %v", user, err)
	}

	// Idempotency
	err = s.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser(%q) error: %v", user, err)
	}
}

func TestAddRecentTracks(t *testing.T) {
	s := createTestDb(t)
	defer s.Close()

	user := "testuser"
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tracks := []TrackImport{
		{
			Artist:    "Test Artist",
			Album:     "Test Album",
			TrackName: "Test Track",
			DateUTS:   "1600000000",
		},
	}

	err := s.AddRecentTracks(user, tracks)
	if err != nil {
		t.Fatalf("AddRecentTracks failed: %v", err)
	}

	// Verify data was inserted
	row := s.db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("querying count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 listen, got %d", count)
	}

	// Test idempotent insert (same data)
	err = s.AddRecentTracks(user, tracks)
	if err != nil {
		t.Fatalf("AddRecentTracks (repeat) failed: %v", err)
	}
	row = s.db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("querying count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 listen after repeat, got %d", count)
	}
}

func TestGetLatestListenWithBadDate(t *testing.T) {
	s := createTestDb(t)
	defer s.Close()

	user := "reproUser"
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Manually insert bad date to simulate legacy data
	// We need to bypass AddRecentTracks to insert bad date if validation existed, 
	// but AddRecentTracks takes string so we can use it.
	
	// Track 1: Bad Date
	tracks := []TrackImport{
		{
			Artist:    "Artist",
			Album:     "Album",
			TrackName: "Track",
			DateUTS:   "0001-01-01T00:00:00Z", // Text date
		},
	}
	if err := s.AddRecentTracks(user, tracks); err != nil {
		t.Fatalf("AddRecentTracks bad date: %v", err)
	}

	// Track 2: Good Date (Unix Timestamp)
	// 1593490750 = 2020-06-30...
	tracks[0].DateUTS = "1593490750"
	if err := s.AddRecentTracks(user, tracks); err != nil {
		t.Fatalf("AddRecentTracks good date: %v", err)
	}

	// Attempt to get latest listen
	date, err := s.GetLatestListen(user)
	if err != nil {
		t.Fatalf("GetLatestListen failed: %v", err)
	}

	// Check if date is the good date
	if date.Year() != 2020 {
		t.Errorf("Expected year 2020, got %v", date)
	}
}

func TestTagUpdates(t *testing.T) {
	s := createTestDb(t)
	defer s.Close()

	// Setup: Add listen for artist/album
	user := "testuser"
	s.CreateUser(user)
	
	// Insert 11 listens to trigger "COUNT(*) > 10" check
	tracks := []TrackImport{}
	for i := 0; i < 11; i++ {
		tracks = append(tracks, TrackImport{
			Artist:    "The Beatles",
			Album:     "Abbey Road",
			TrackName: "Come Together",
			DateUTS:   "1600000000", // timestamp doesn't matter for count
		})
	}
	// Need unique timestamps for Listen unique constraint? 
	// createListen checks (user, track, date). Same track/user needs diff date.
	// Ah, createListen uses SELECT id... IF exists return nil.
	// So we need different dates.
	for i := range tracks {
		tracks[i].DateUTS =  "16000000" + string(rune(i+48)) // Hacky but works for string
	}
	
	s.AddRecentTracks(user, tracks)

	// Check Artists Needing Update
	artists, err := s.GetArtistsNeedingTagUpdate(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetArtistsNeedingTagUpdate: %v", err)
	}
	if len(artists) != 1 || artists[0] != "The Beatles" {
		t.Errorf("Expected [The Beatles], got %v", artists)
	}

	// Update Tags
	err = s.SaveArtistTags("The Beatles", []string{"Classic Rock"}, []int{100})
	if err != nil {
		t.Fatalf("SaveArtistTags: %v", err)
	}

	// Check again - should be empty (recently updated)
	artists, err = s.GetArtistsNeedingTagUpdate(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetArtistsNeedingTagUpdate 2: %v", err)
	}
	if len(artists) != 0 {
		t.Errorf("Expected [], got %v", artists)
	}

	// Check Albums Needing Update
	albums, err := s.GetAlbumsNeedingTagUpdate(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetAlbumsNeedingTagUpdate: %v", err)
	}
	if len(albums) != 1 || albums[0].Name != "Abbey Road" {
		t.Errorf("Expected [Abbey Road], got %v", albums)
	}

	// Update Album Tags
	err = s.SaveAlbumTags("The Beatles", "Abbey Road", []string{"Masterpiece"}, []int{100})
	if err != nil {
		t.Fatalf("SaveAlbumTags: %v", err)
	}

	// Check again
	albums, err = s.GetAlbumsNeedingTagUpdate(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetAlbumsNeedingTagUpdate 2: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("Expected [], got %v", albums)
	}
}
