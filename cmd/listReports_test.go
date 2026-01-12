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

func TestListReports(t *testing.T) {
	db, dbPath := createTestDb(t)

	user := "testuser"
	err := addReport(dbPath, "report1", user, "user@example.com", 1, []string{"top-albums"})
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}

	err = listReports(dbPath, user)
	if err != nil {
		t.Fatalf("listReports() error: %w", err)
	}

	// Verify it handles no reports gracefully
	_, err = db.Exec("DELETE FROM Report")
	if err != nil {
		t.Fatalf("failed to clear reports: %v", err)
	}

	err = listReports(dbPath, user)
	if err != nil {
		t.Fatalf("listReports() should not fail even with no reports: %v", err)
	}
}
