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
	"testing"
)

func TestSendReports(t *testing.T) {
	_, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	err = addReport(getTestDbPath(), "test report", "testuser", "testuser@gmail.com", 1, []string{"top-albums", "top-artists"})
	err = addReport(getTestDbPath(), "other test report", "other user", "otheruser@gmail.com", 1, []string{"new-albums", "new-artists"})
	if err != nil {
		t.Fatalf("addReport() error: %w", err)
	}

	config := SendReportsConfig{
		DbPath: getTestDbPath(),
		From:   "from@from.com",
		DryRun: true,
	}
	err = sendReports(config)
	if err != nil {
		t.Fatalf("sendReports() error: %w", err)
	}

	err = sendReports(config)
	if err != nil {
		t.Fatalf("sendReports() error on second run: %w", err)
	}
}
