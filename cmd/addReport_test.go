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
	"strings"
	"testing"
	"time"
)

func TestAddReport(t *testing.T) {
	_, dbPath := createTestDb(t)

	err := addReport(dbPath, "test report", "testuser", "testuser@gmail.com", 1, 0, "", []string{"top-albums", "top-artists"}, nil)
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}
}

func TestAddReportInvalidAction(t *testing.T) {
	invalidAction := "not-real"

	_, dbPath := createTestDb(t)

	err := addReport(dbPath, "test report", "testuser", "testuser@gmail.com", 1, 0, "", []string{invalidAction}, nil)
	if err == nil {
		t.Fatalf("addReport should have failed with invalid action")
	}
	if !strings.Contains(err.Error(), invalidAction) {
		t.Fatalf("Should have error with invalid action (%q): %w", invalidAction, err)
	}
}

func TestAddReportRelativeFirstRun(t *testing.T) {
	db, dbPath := createTestDb(t)

	// Test 30d
	err := addReport(dbPath, "relative report", "testuser", "testuser@gmail.com", 1, 0, "30d", []string{"top-albums"}, nil)
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}

	var nextRun time.Time
	err = db.QueryRow("SELECT next_run FROM Report WHERE name = ?", "relative report").Scan(&nextRun)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	expected := time.Now().AddDate(0, 0, 30)
	// Allow 5 minutes difference
	if nextRun.Sub(expected) > 5*time.Minute || nextRun.Sub(expected) < -5*time.Minute {
		t.Errorf("Expected next_run around %v, got %v", expected, nextRun)
	}

	// Test absolute date (regression)
	absoluteDate := "2025-01-01"
	err = addReport(dbPath, "absolute report", "testuser", "testuser@gmail.com", 1, 0, absoluteDate, []string{"top-albums"}, nil)
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}

	err = db.QueryRow("SELECT next_run FROM Report WHERE name = ?", "absolute report").Scan(&nextRun)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	expectedAbs, _ := time.Parse("2006-01-02", absoluteDate)
	if !nextRun.Equal(expectedAbs) {
		t.Errorf("Expected next_run %v, got %v", expectedAbs, nextRun)
	}
}
