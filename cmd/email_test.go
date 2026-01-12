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
	"strconv"
	"strings"
	"testing"
	"time"
)

func createListenForDate(db *sql.DB, user string, t time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("creating transaction: %w", err)
	}

	artist := "Dummy Artist"
	album := "Dummy Album"
	track := "Dummy Track"

	if err := createArtist(tx, artist); err != nil {
		return err
	}
	if err := createAlbum(tx, artist, album); err != nil {
		return err
	}
	trackID, err := createTrack(tx, artist, album, track)
	if err != nil {
		return err
	}

	err = createListen(tx, user, trackID, strconv.FormatInt(t.Unix(), 10))
	if err != nil {
		return fmt.Errorf("createListen(): %w", err)
	}
	tx.Commit()

	return nil
}

func TestGenerateEmailContent(t *testing.T) {
	dbPath := getTestDbPath()
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}
	defer db.Close()

	user := "testuser"
	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser: %w", err)
	}

	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)
	listenTime := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("creating transaction: %w", err)
	}
	artist := "Test Artist"
	if err := createArtist(tx, artist); err != nil {
		t.Fatalf("createArtist: %w", err)
	}
	if err := createAlbum(tx, artist, "Test Album"); err != nil {
		t.Fatalf("createAlbum: %w", err)
	}
	trackID, err := createTrack(tx, artist, "Test Album", "Test Track")
	if err != nil {
		t.Fatalf("createTrack: %w", err)
	}
	if err := createListen(tx, user, trackID, strconv.FormatInt(listenTime.Unix(), 10)); err != nil {
		t.Fatalf("createListen: %w", err)
	}
	tx.Commit()

	config := SendEmailConfig{
		DbPath:     dbPath,
		User:       user,
		ReportName: "Monthly Report",
		Start:      start,
		End:        end,
	}

	// Use actual analyzers for integration testing
	actions := []Analyser{
		TopArtistsAnalyzer{}.SetConfig(AnalyserConfig{20, 0}),
	}

	subject, body, err := generateEmailContent(config, actions)
	if err != nil {
		t.Fatalf("generateEmailContent failed: %v", err)
	}

	// Verify Subject
	expectedSubject := fmt.Sprintf("Listening report for %s %s to %s: %s", user, start.Format("2006-01-02"), end.Format("2006-01-02"), config.ReportName)
	if subject != expectedSubject {
		t.Errorf("Subject mismatch.\nGot: %s\nWant: %s", subject, expectedSubject)
	}

	// Verify Body content
	if !strings.Contains(body, "Top artists") {
		t.Error("Body missing analyzer name 'Top artists'")
	}
	if !strings.Contains(body, "<h2>Top artists for testuser 2023-01-01 to 2023-02-01:</h2>") {
		t.Error("Body missing correct header with dates")
	}
	// Since we added a listen, we expect the table
	if !strings.Contains(body, "<table>") {
		t.Error("Body missing table")
	}
	if strings.Contains(body, "No listens found") {
		t.Error("Body incorrectly reports 'No listens found'")
	}
}

func TestGenerateEmailContentNoData(t *testing.T) {
	dbPath := getTestDbPath()
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}
	defer db.Close()

	user := "testuser"
	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser: %w", err)
	}

	// Range with no listens
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)

	config := SendEmailConfig{
		DbPath:     dbPath,
		User:       user,
		ReportName: "",
		Start:      start,
		End:        end,
	}

	actions := []Analyser{
		TopArtistsAnalyzer{}.SetConfig(AnalyserConfig{20, 0}),
	}

	subject, body, err := generateEmailContent(config, actions)
	if err != nil {
		t.Fatalf("generateEmailContent failed: %v", err)
	}

	// Verify Subject (no suffix)
	expectedSubject := fmt.Sprintf("Listening report for %s %s to %s", user, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if subject != expectedSubject {
		t.Errorf("Subject mismatch.\nGot: %s\nWant: %s", subject, expectedSubject)
	}

	// Verify Body
	if !strings.Contains(body, "No listens found") {
		t.Error("Body missing 'No listens found' message")
	}
	if strings.Contains(body, "<table>") {
		t.Error("Body should not contain a table")
	}
}
