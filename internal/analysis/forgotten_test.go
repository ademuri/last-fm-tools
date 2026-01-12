package analysis

import (
	"testing"
	"time"
)

func TestGetForgottenArtists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)

	// Artist A: 100 listens, last listen 2 years ago (Obsession)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist A")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'Track 1', 'Artist A', 'Album A1')")
	for i := 0; i < ThresholdArtistObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, twoYearsAgo.Unix()-int64(i))
	}

	// Artist B: 100 listens, last listen today (Not Forgotten)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist B")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (2, 'Track 2', 'Artist B', 'Album B1')")
	for i := 0; i < ThresholdArtistObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, now.Unix()-int64(i))
	}

	// Artist C: 15 listens, last listen 2 years ago (Moderate)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist C")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (3, 'Track 3', 'Artist C', 'Album C1')")
	// ThresholdArtistModerate is 10, so 15 is safely Moderate (and < Strong 30)
	for i := 0; i < 15; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 3, ?)", user, twoYearsAgo.Unix()-int64(i))
	}

	// Artist D: 5 listens, last listen 2 years ago (Ignored - too few)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist D")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (4, 'Track 4', 'Artist D', 'Album D1')")
	// ThresholdArtistModerate is 10, so 5 is ignored
	for i := 0; i < 5; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 4, ?)", user, twoYearsAgo.Unix()-int64(i))
	}

	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		MinArtistScrobbles: ThresholdArtistModerate,
		MinAlbumScrobbles:  ThresholdAlbumModerate,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}

	// Check Obsession (Artist A)
	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession artist, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Artist != "Artist A" {
		t.Errorf("expected Artist A in obsession, got %s", results[BandObsession][0].Artist)
	}

	// Check Strong (None)
	if len(results[BandStrong]) != 0 {
		t.Errorf("expected 0 strong artists, got %d", len(results[BandStrong]))
	}

	// Check Moderate (Artist C)
	if len(results[BandModerate]) != 1 {
		t.Errorf("expected 1 moderate artist, got %d", len(results[BandModerate]))
	} else if results[BandModerate][0].Artist != "Artist C" {
		t.Errorf("expected Artist C in moderate, got %s", results[BandModerate][0].Artist)
	}

	// Ensure Artist B and D are not present
	for _, band := range results {
		for _, a := range band {
			if a.Artist == "Artist B" {
				t.Error("Artist B should not be forgotten")
			}
			if a.Artist == "Artist D" {
				t.Error("Artist D should be filtered out by threshold")
			}
		}
	}
}

func TestGetForgottenAlbums(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)

	// Artist A, Album A1: 50 listens, old (Obsession)
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist A", "Album A1")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'Track 1', 'Artist A', 'Album A1')")
	for i := 0; i < ThresholdAlbumObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, twoYearsAgo.Unix()-int64(i))
	}

	// Artist A, Album A2: 5 listens, old (Moderate)
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist A", "Album A2")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (2, 'Track 2', 'Artist A', 'Album A2')")
	// ThresholdAlbumModerate is 5
	for i := 0; i < ThresholdAlbumModerate; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, twoYearsAgo.Unix()-int64(i))
	}

	// Artist B, Album B1: 50 listens, recent (Not Forgotten)
	db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", "Artist B", "Album B1")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (3, 'Track 3', 'Artist B', 'Album B1')")
	for i := 0; i < ThresholdAlbumObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 3, ?)", user, now.Unix()-int64(i))
	}

	config := ForgottenConfig{
		LastListenAfter:    time.Unix(0, 0),
		LastListenBefore:   now.AddDate(0, 0, -90),
		MinArtistScrobbles: ThresholdArtistModerate,
		MinAlbumScrobbles:  ThresholdAlbumModerate,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenAlbums(db, user, config)
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
	now := time.Now()

	// Artist A: Last listen 5 years ago (Too old if we set after=3 years ago)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist A")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'T1', 'Artist A', 'A1')")
	for i := 0; i < ThresholdArtistObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 1, ?)", user, now.AddDate(-5, 0, 0).Unix()-int64(i))
	}

	// Artist B: Last listen 2 years ago (Should be found)
	db.Exec("INSERT INTO Artist (name) VALUES (?)", "Artist B")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (2, 'T2', 'Artist B', 'B1')")
	for i := 0; i < ThresholdArtistObsession; i++ {
		db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, 2, ?)", user, now.AddDate(-2, 0, 0).Unix()-int64(i))
	}

	config := ForgottenConfig{
		LastListenAfter:    now.AddDate(-3, 0, 0), // Only listen since 3 years ago
		LastListenBefore:   now.AddDate(-1, 0, 0), // But not in last 1 year
		MinArtistScrobbles: 10,
		ResultsPerBand:     10,
		SortBy:             "dormancy",
	}

	results, err := GetForgottenArtists(db, user, config)
	if err != nil {
		t.Fatalf("GetForgottenArtists failed: %v", err)
	}

	if len(results[BandObsession]) != 1 {
		t.Errorf("expected 1 obsession artist, got %d", len(results[BandObsession]))
	} else if results[BandObsession][0].Artist != "Artist B" {
		t.Errorf("expected Artist B, got %s", results[BandObsession][0].Artist)
	}
}
