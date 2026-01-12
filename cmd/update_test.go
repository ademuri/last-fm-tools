/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/ademuri/lastfm-go/lastfm"
)

func createTestDb(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "lastfm.db")

	db, err := createDatabase(dbPath)
	if err != nil {
		t.Fatalf("createDatabase(%s) error: %v", dbPath, err)
	}
	if db == nil {
		t.Fatalf("createDatabase(%s) returned nil", dbPath)
	}

	return db, dbPath
}

func TestCreateDatabaseAndData(t *testing.T) {
	db, _ := createTestDb(t)
	defer db.Close()

	user := "testuser"
	err := createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %v", user, err)
	}

	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %v", user, err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}

	artist := "The Beatles"

	err = createArtist(tx, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %v", artist, err)
	}

	err = createArtist(tx, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %v", artist, err)
	}

	album := "White Album"
	err = createAlbum(tx, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %v", artist, album, err)
	}

	err = createAlbum(tx, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %v", artist, album, err)
	}

	track := "Ob-La-Di, Ob-La-Da"
	trackID, err := createTrack(tx, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %v", artist, album, track, err)
	}

	_, err = createTrack(tx, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %v", artist, album, track, err)
	}

	datetime := "1"
	err = createListen(tx, user, trackID, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %v", trackID, datetime, err)
	}

	err = createListen(tx, user, trackID, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %v", trackID, datetime, err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("commiting transaction: %v", err)
	}
}

func TestGetLatestListenWithBadDate(t *testing.T) {
	db, _ := createTestDb(t)
	defer db.Close()

	user := "reproUser"
	err := createUser(db, user)
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin: %v", err)
	}

	createArtist(tx, "Artist")
	createAlbum(tx, "Artist", "Album")
	trackID, err := createTrack(tx, "Artist", "Album", "Track")
	if err != nil {
		t.Fatalf("createTrack: %v", err)
	}

	// Insert bad date (Text, which sorts > Integer in SQLite)
	badDate := "0001-01-01T00:00:00Z"
	err = createListen(tx, user, trackID, badDate)
	if err != nil {
		t.Fatalf("createListen failed: %v", err)
	}

	// Insert good date (Unix Timestamp)
	// 1593490750 = 2020-06-30...
	goodDate := "1593490750"
	err = createListen(tx, user, trackID, goodDate)
	if err != nil {
		t.Fatalf("createListen failed: %v", err)
	}

	tx.Commit()

	// Attempt to get latest listen
	date, err := getLatestListen(db, user)
	if err != nil {
		t.Fatalf("getLatestListen failed: %v", err)
	}

	// Check if date is the good date
	// 1593490750
	if date.Year() != 2020 {
		t.Errorf("Expected year 2020, got %v", date)
	}
}

func TestInsertRecentTracks(t *testing.T) {
	db, _ := createTestDb(t)
	defer db.Close()

	user := "testuser"
	err := createUser(db, user)
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}

	recentTracks := lastfm.UserGetRecentTracks{
		Page:       1,
		TotalPages: 1,
		Tracks: []struct {
			NowPlaying string `xml:"nowplaying,attr,omitempty"`
			Artist     struct {
				Name string `xml:",chardata"`
				Mbid string `xml:"mbid,attr"`
			} `xml:"artist"`
			Name       string `xml:"name"`
			Streamable string `xml:"streamable"`
			Mbid       string `xml:"mbid"`
			Album      struct {
				Name string `xml:",chardata"`
				Mbid string `xml:"mbid,attr"`
			} `xml:"album"`
			Url    string `xml:"url"`
			Images []struct {
				Size string `xml:"size,attr"`
				Url  string `xml:",chardata"`
			} `xml:"image"`
			Date struct {
				Uts  string `xml:"uts,attr"`
				Date string `xml:",chardata"`
			} `xml:"date"`
		}{
			{
				Name: "Test Track",
				Artist: struct {
					Name string `xml:",chardata"`
					Mbid string `xml:"mbid,attr"`
				}{
					Name: "Test Artist",
				},
				Album: struct {
					Name string `xml:",chardata"`
					Mbid string `xml:"mbid,attr"`
				}{
					Name: "Test Album",
				},
				Date: struct {
					Uts  string `xml:"uts,attr"`
					Date string `xml:",chardata"`
				}{
					Uts: "1600000000",
				},
			},
		},
	}

	err = insertRecentTracks(db, user, recentTracks)
	if err != nil {
		t.Fatalf("insertRecentTracks failed: %v", err)
	}

	// Verify data was inserted
	row := db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("querying count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 listen, got %d", count)
	}

	// Test idempotent insert (same data)
	err = insertRecentTracks(db, user, recentTracks)
	if err != nil {
		t.Fatalf("insertRecentTracks (repeat) failed: %v", err)
	}
	row = db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("querying count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 listen after repeat, got %d", count)
	}
}