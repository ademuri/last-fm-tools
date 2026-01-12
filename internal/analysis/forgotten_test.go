package analysis

import (
	"fmt"
	"testing"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
)

// Better helper that handles everything
func setupArtistAndListens(t *testing.T, db *store.Store, user string, id int, artist, album string, count int, lastListen time.Time) {
	t.Helper()
	
	// We use AddRecentTracks to insert data.
	// It handles artist, album, track creation.
	// We need to generate 'count' tracks with timestamps ending at lastListen.
	
	var tracks []store.TrackImport
	for i := 0; i < count; i++ {
		ts := lastListen.Add(time.Duration(-i) * time.Minute)
		tracks = append(tracks, store.TrackImport{
			Artist:    artist,
			Album:     album,
			TrackName: fmt.Sprintf("Track %s %s", artist, album), // Simple track name reuse
			DateUTS:   fmt.Sprintf("%d", ts.Unix()),
		})
	}
	
	// AddRecentTracks batches.
	if err := db.AddRecentTracks(user, tracks); err != nil {
		t.Fatalf("AddRecentTracks failed: %v", err)
	}
	
	// Note: We ignored 'id' param because AddRecentTracks auto-generates IDs. 
	// This should be fine as long as tests don't rely on specific IDs.
}

func TestGetForgottenArtists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user) // Ensure user exists

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	twoYearsAgo := now.AddDate(-2, 0, 0)

	setupArtistAndListens(t, db, user, 1, "Artist A", "Album A1", ThresholdArtistObsession, twoYearsAgo) // Obsession
	setupArtistAndListens(t, db, user, 2, "Artist B", "Album B1", ThresholdArtistObsession, now)         // Recent
	setupArtistAndListens(t, db, user, 3, "Artist C", "Album C1", ThresholdArtistModerate, twoYearsAgo)  // Moderate
	setupArtistAndListens(t, db, user, 4, "Artist D", "Album D1", 5, twoYearsAgo)                        // Ignore

	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		FirstListenAfter:   time.Unix(0, 0),
		FirstListenBefore:  now.AddDate(1, 0, 0),
		MinArtistScrobbles: ThresholdArtistModerate,
		MinAlbumScrobbles:  ThresholdAlbumModerate,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}

	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession artist, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Artist != "Artist A" {
		t.Errorf("expected Artist A in obsession, got %s", results[BandObsession][0].Artist)
	}

	if len(results[BandStrong]) != 0 {
		t.Errorf("expected 0 strong artists, got %d", len(results[BandStrong]))
	}

	if len(results[BandModerate]) != 1 {
		t.Errorf("expected 1 moderate artist, got %d", len(results[BandModerate]))
	} else if results[BandModerate][0].Artist != "Artist C" {
		t.Errorf("expected Artist C in moderate, got %s", results[BandModerate][0].Artist)
	}
}

func TestGetForgottenAlbums(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user)

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	twoYearsAgo := now.AddDate(-2, 0, 0)

	setupArtistAndListens(t, db, user, 1, "Artist A", "Album A1", ThresholdAlbumObsession, twoYearsAgo)
	setupArtistAndListens(t, db, user, 2, "Artist A", "Album A2", ThresholdAlbumModerate, twoYearsAgo)
	setupArtistAndListens(t, db, user, 3, "Artist B", "Album B1", ThresholdAlbumObsession, now)

	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		FirstListenAfter:   time.Unix(0, 0),
		FirstListenBefore:  now.AddDate(1, 0, 0),
		MinArtistScrobbles: ThresholdArtistModerate,
		MinAlbumScrobbles:  ThresholdAlbumModerate,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenAlbums(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenAlbums failed: %v", err)
	}

	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession album, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Album != "Album A1" {
		t.Errorf("expected Album A1 in obsession, got %s", results[BandObsession][0].Album)
	}

	if len(results[BandModerate]) != 1 {
		t.Errorf("expected 1 moderate album, got %d", len(results[BandModerate]))
	} else if results[BandModerate][0].Album != "Album A2" {
		t.Errorf("expected Album A2 in moderate, got %s", results[BandModerate][0].Album)
	}
}

func TestGetForgottenArtistsWithDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user)

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Artist A: Last listen 5 years ago
	setupArtistAndListens(t, db, user, 1, "Artist A", "A1", ThresholdArtistObsession, now.AddDate(-5, 0, 0))

	// Artist B: Last listen 2 years ago
	setupArtistAndListens(t, db, user, 2, "Artist B", "B1", ThresholdArtistObsession, now.AddDate(-2, 0, 0))

	config := ForgottenConfig{
		LastListenAfter:    now.AddDate(-3, 0, 0), // Only listen since 3 years ago
		LastListenBefore:   now.AddDate(-1, 0, 0), // But not in last 1 year
		FirstListenAfter:   time.Unix(0, 0),
		FirstListenBefore:  now.AddDate(1, 0, 0),
		MinArtistScrobbles: 10,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}

	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession artist, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Artist != "Artist B" {
		t.Errorf("expected Artist B, got %s", results[BandObsession][0].Artist)
	}
}

func TestGetForgottenArtistsWithFirstListenDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user)

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Artist A: First listen 10 years ago, last listen 2 years ago
	setupArtistAndListens(t, db, user, 1, "Artist A", "A1", ThresholdArtistObsession, now.AddDate(-2, 0, 0))
	// Add one extra listen 10 years ago manually via AddRecentTracks
	ts := now.AddDate(-10, 0, 0)
	db.AddRecentTracks(user, []store.TrackImport{{
		Artist: "Artist A", Album: "A1", TrackName: "Old Track", DateUTS: fmt.Sprintf("%d", ts.Unix()),
	}})

	// Artist B: First listen 3 years ago, last listen 2 years ago
	setupArtistAndListens(t, db, user, 2, "Artist B", "B1", ThresholdArtistObsession, now.AddDate(-2, 0, 0))
	ts = now.AddDate(-3, 0, 0)
	db.AddRecentTracks(user, []store.TrackImport{{
		Artist: "Artist B", Album: "B1", TrackName: "Old Track B", DateUTS: fmt.Sprintf("%d", ts.Unix()),
	}})

	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		FirstListenAfter:   now.AddDate(-5, 0, 0),
		FirstListenBefore:  now.AddDate(-2, 0, 0), // First listen must be between 5 and 2 years ago
		MinArtistScrobbles: 10,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}

	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession artist, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Artist != "Artist B" {
		t.Errorf("expected Artist B, got %s", results[BandObsession][0].Artist)
	}
}

func TestSortingAndLimits(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	db.CreateUser(user)

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Artist A: 150 scrobbles, last listen 5 years ago
	setupArtistAndListens(t, db, user, 1, "Artist A", "A1", 150, now.AddDate(-5, 0, 0))
	
	// Artist B: 200 scrobbles, last listen 2 years ago
	setupArtistAndListens(t, db, user, 2, "Artist B", "B1", 200, now.AddDate(-2, 0, 0))

	// Artist C: 120 scrobbles, last listen 3 years ago
	setupArtistAndListens(t, db, user, 3, "Artist C", "C1", 120, now.AddDate(-3, 0, 0))

	// All are Obsession (>100)

	// 1. Sort by Dormancy (default): Oldest LastListen first
	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		FirstListenAfter:   time.Unix(0, 0),
		FirstListenBefore:  now.AddDate(1, 0, 0),
		MinArtistScrobbles: 10,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}
	
	list := results[BandObsession]
	if len(list) != 3 {
		t.Fatalf("Expected 3 artists, got %d", len(list))
	}
	if list[0].Artist != "Artist A" || list[1].Artist != "Artist C" || list[2].Artist != "Artist B" {
		t.Errorf("Dormancy sort failed. Got: %s, %s, %s", list[0].Artist, list[1].Artist, list[2].Artist)
	}

	// 2. Sort by Listens: Most scrobbles first
	config.SortBy = "listens"
	results, err = GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}
	list = results[BandObsession]
	if list[0].Artist != "Artist B" || list[1].Artist != "Artist A" || list[2].Artist != "Artist C" {
		t.Errorf("Listens sort failed. Got: %s, %s, %s", list[0].Artist, list[1].Artist, list[2].Artist)
	}

	// 3. Limits
	config.ResultsPerBand = 2
	results, err = GetForgottenArtists(db, user, config, now)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}
	list = results[BandObsession]
	if len(list) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(list))
	}
	// With "listens" sort, should be B and A
	if list[0].Artist != "Artist B" || list[1].Artist != "Artist A" {
		t.Errorf("Limit failed. Got: %s, %s", list[0].Artist, list[1].Artist)
	}
}
