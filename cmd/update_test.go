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

import "testing"

const (
	user = "testuser"
)

func TestCreateDatabaseAndData(t *testing.T) {
	db, err := createDatabase()
	if err != nil {
		t.Fatalf("createDatabase() error: %w", err)
	}

	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %w", user, err)
	}

	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %w", user, err)
	}

	artist := "The Beatles"
	err = createArtist(db, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %w", artist, err)
	}

	err = createArtist(db, artist)
	if err != nil {
		t.Fatalf("createArtist(%q) error: %w", artist, err)
	}

	album := "White Album"
	err = createAlbum(db, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %w", artist, album, err)
	}

	err = createAlbum(db, artist, album)
	if err != nil {
		t.Fatalf("createAlbum(%q, %q) error: %w", artist, album, err)
	}

	track := "Ob-La-Di, Ob-La-Da"
	track_id, err := createTrack(db, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %w", artist, album, track, err)
	}

	_, err = createTrack(db, artist, album, track)
	if err != nil {
		t.Fatalf("createTrack(%q, %q, %q) error: %w", artist, album, track, err)
	}

	datetime := "1"
	err = createListen(db, track_id, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %w", track_id, datetime, err)
	}

	err = createListen(db, track_id, datetime)
	if err != nil {
		t.Fatalf("createListen(%q, %q) error: %w", track_id, datetime, err)
	}
}