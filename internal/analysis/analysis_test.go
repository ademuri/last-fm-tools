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

	// 3. Tracks
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'Track 1', 'Artist A', 'Album A1')")
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (2, 'Track 2', 'Artist B', 'Album B1')")

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

	// 5. Tags
	db.Exec("INSERT INTO Tag (name) VALUES (?)", "Rock")
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist A", "Rock", 100)
	db.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", "Artist B", "Rock", 50)

	report, err := GenerateReport(db, user)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report.Metadata.TotalScrobbles != 15 {
		t.Errorf("expected 15 scrobbles, got %d", report.Metadata.TotalScrobbles)
	}

	if len(report.CurrentTaste.TopArtists) == 0 {
		t.Errorf("expected at least one current artist")
	} else if report.CurrentTaste.TopArtists[0].Name != "Artist A" {
		t.Errorf("expected Artist A to be top current artist, got %s", report.CurrentTaste.TopArtists[0].Name)
	}

	if len(report.HistoricalBaseline.TopArtists) == 0 {
		t.Errorf("expected at least one historical artist")
	} else if report.HistoricalBaseline.TopArtists[0].Name != "Artist B" {
		t.Errorf("expected Artist B to be top historical artist, got %s", report.HistoricalBaseline.TopArtists[0].Name)
	}
}

func TestGetPeakYears(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := "testuser"
	artist := "Artist A"
	db.Exec("INSERT INTO Track (id, name, artist, album) VALUES (1, 'T', ?, 'A')", artist)

	// 10 listens in 2020, 10 in 2021, 10 in 2022
	// Total 30. 80% is 24.
	// Any 3 year range will have 30.
	// Shortest range containing 24? 
	// 2020: 10
	// 2021: 10
	// 2022: 10
	// Range 2020-2022 has 30. 2020-2021 has 20. 2021-2022 has 20.
	// So 2020-2022 is shortest range >= 24.

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
