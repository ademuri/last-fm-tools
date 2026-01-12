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
)

func TestAddReport(t *testing.T) {
	_, dbPath := createTestDb(t)

	err := addReport(dbPath, "test report", "testuser", "testuser@gmail.com", 1, []string{"top-albums", "top-artists"})
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}
}

func TestAddReportInvalidAction(t *testing.T) {
	invalidAction := "not-real"

	_, dbPath := createTestDb(t)

	err := addReport(dbPath, "test report", "testuser", "testuser@gmail.com", 1, []string{invalidAction})
	if err == nil {
		t.Fatalf("addReport should have failed with invalid action")
	}
	if !strings.Contains(err.Error(), invalidAction) {
		t.Fatalf("Should have error with invalid action (%q): %w", invalidAction, err)
	}
}
