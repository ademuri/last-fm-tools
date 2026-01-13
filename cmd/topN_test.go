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
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestPrintTopNWithTags(t *testing.T) {
	// Setup temporary database using helper from update_test.go (same package)
	db, dbPath := createTestDb(t)
	defer db.Close()

	user := "testuser"
	viper.Set("user", user)

	// Populate data
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin: %v", err)
	}

	artist := "The Beatles"
album := "Abbey Road"
track := "Come Together"

	createArtist(tx, artist)
	createAlbum(tx, artist, album)
	trackID, _ := createTrack(tx, artist, album, track)

	// Create a listen in the past
	listenDate := time.Now().AddDate(0, -1, 0).Unix() // 1 month ago
	_, err = tx.Exec("INSERT INTO Listen (user, track, date) VALUES (?, ?, ?)", user, trackID, listenDate)
	if err != nil {
		t.Fatalf("inserting listen: %v", err)
	}

	// Add tags
	tags := []string{"rock", "classic rock", "british", "60s"}
	for _, tag := range tags {
		_, err = tx.Exec("INSERT OR IGNORE INTO Tag (name) VALUES (?)", tag)
		if err != nil {
			t.Fatalf("inserting tag %s: %v", tag, err)
		}
	}

	// Link tags to artist (count matters for ordering)
	// rock: 100, classic rock: 90, british: 80, 60s: 70
	_, err = tx.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", artist, "rock", 100)
	_, err = tx.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", artist, "classic rock", 90)
	_, err = tx.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", artist, "british", 80)
	_, err = tx.Exec("INSERT INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", artist, "60s", 70)

	// Link tags to album
	// rock: 50, classic rock: 40
	_, err = tx.Exec("INSERT INTO AlbumTag (artist, album, tag, count) VALUES (?, ?, ?, ?)", artist, album, "rock", 50)
	_, err = tx.Exec("INSERT INTO AlbumTag (artist, album, tag, count) VALUES (?, ?, ?, ?)", artist, album, "classic rock", 40)

	tx.Commit()

	// Configure limits
	limitArtists = 10
	limitAlbums = 10
	limitTracks = 10
	limitTags = 2 // Should show top 2 tags

	// Run printTopN
	var out bytes.Buffer
	// Range: 2 months ago to now
	startTime := time.Now().AddDate(0, -2, 0)
	endTime := time.Now()

	err = printTopN(&out, dbPath, startTime, endTime, limitArtists, limitAlbums, limitTracks, limitTags)
	if err != nil {
		t.Fatalf("printTopN failed: %v", err)
	}

	output := out.String()

	// Assertions
	// Check Artist Tags
	expectedArtistTags := "[rock, classic rock]"
	if !strings.Contains(output, expectedArtistTags) {
		t.Errorf("Output missing expected artist tags %q. Got:\n%s", expectedArtistTags, output)
	}
	// Ensure 3rd tag is NOT present
	if strings.Contains(output, "british") {
		t.Errorf("Output should not contain 3rd tag 'british' when limitTags=2")
	}

	// Check Album Tags
	expectedAlbumTags := "[rock, classic rock]"
	if !strings.Contains(output, expectedAlbumTags) {
		t.Errorf("Output missing expected album tags %q. Got:\n%s", expectedAlbumTags, output)
	}
}
