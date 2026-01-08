/*
Copyright 2026 Google LLC

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
	"testing"
)

func TestDeleteReport(t *testing.T) {
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	user := "testuser"
	reportName := "reportToDelete"
	email := "user@example.com"
	err = addReport(getTestDbPath(), reportName, user, email, 1, []string{"top-albums"})
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}

	// Verify it exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM Report WHERE user = ? AND name = ?", user, reportName).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 report, got %d", count)
	}

	// Delete it
	err = deleteReport(getTestDbPath(), user, reportName, email)
	if err != nil {
		t.Fatalf("deleteReport() error: %w", err)
	}

	// Verify it's gone
	err = db.QueryRow("SELECT COUNT(*) FROM Report WHERE user = ? AND name = ?", user, reportName).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reports, got %d", count)
	}
}

func TestDeleteReportWithDuplicateName(t *testing.T) {
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	user := "testuser"
	reportName := "duplicateReport"
	email1 := "user1@example.com"
	email2 := "user2@example.com"

	err = addReport(getTestDbPath(), reportName, user, email1, 1, []string{"top-albums"})
	if err != nil {
		t.Fatalf("addReport() 1 error: %w", err)
	}
	err = addReport(getTestDbPath(), reportName, user, email2, 1, []string{"top-artists"})
	if err != nil {
		t.Fatalf("addReport() 2 error: %w", err)
	}

	// Delete first one
	err = deleteReport(getTestDbPath(), user, reportName, email1)
	if err != nil {
		t.Fatalf("deleteReport() error: %w", err)
	}

	// Verify first is gone, second remains
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM Report WHERE user = ? AND name = ? AND email = ?", user, reportName, email1).Scan(&count)
	if err != nil {
		t.Fatalf("query 1 failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected report 1 to be gone")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM Report WHERE user = ? AND name = ? AND email = ?", user, reportName, email2).Scan(&count)
	if err != nil {
		t.Fatalf("query 2 failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected report 2 to remain")
	}
}

func TestDeleteNonExistentReport(t *testing.T) {
	_, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	user := "testuser"
	err = deleteReport(getTestDbPath(), user, "nonExistent", "any@email.com")
	if err == nil {
		t.Fatalf("deleteReport() should fail for non-existent report")
	}
}
