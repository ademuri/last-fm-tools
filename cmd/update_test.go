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
	"fmt"
	"os"
	"testing"
)

const (
	user = "testuser"
)

func getTestDbPath() string {
	return os.Getenv("TEST_TMPDIR") + "/lastfm.db"
}

func createTestDb() (*sql.DB, error) {
	dbPath := getTestDbPath()
	// Delete the database if it already exists
	if _, err := os.Stat(dbPath); err == nil {
		err = os.Remove(dbPath)
		if err != nil {
			return nil, fmt.Errorf("Deleting previous database: %w", err)
		}
	}

	db, err := createDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("createDatabase(%s) error: %w", dbPath, err)
	}
	if db == nil {
		return nil, fmt.Errorf("createDatabase(%s) returned nil", dbPath)
	}

	return db, nil
}

func TestCreateDatabaseAndData(t *testing.T) {
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	user := "testuser"
	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %w", user, err)
	}

	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %w", user, err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("creating transaction: %w", err)
	}

	artist := "The Beatles"

	err = createArtist(tx, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %w", artist, err)
	}

	err = createArtist(tx, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %w", artist, err)
	}

	album := "White Album"
	err = createAlbum(tx, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %w", artist, album, err)
	}

	err = createAlbum(tx, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %w", artist, album, err)
	}

	track := "Ob-La-Di, Ob-La-Da"
	track_id, err := createTrack(tx, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %w", artist, album, track, err)
	}

	_, err = createTrack(tx, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %w", artist, album, track, err)
	}

	datetime := "1"
	err = createListen(tx, user, track_id, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %w", track_id, datetime, err)
	}

	err = createListen(tx, user, track_id, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %w", track_id, datetime, err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("commiting transaction: %w", err)
	}
}

func TestGetLatestListenWithBadDate(t *testing.T) {
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}
	defer db.Close()

	user := "reproUser"
	err = createUser(db, user)
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
