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
	"os"
	"strings"
	"testing"
)

func TestPrintNewArtistsDatabaseDoesntExist(t *testing.T) {
	err := printNewArtists(os.Getenv("TEST_TMPDIR")+"/invalid.db", 10, []string{"2020-05"})
	if err == nil {
		t.Fatalf("printNewArtists should have errored with no database")
	}
	if !strings.Contains(err.Error(), "doesn't exist") {
		t.Fatalf("printNewArtists should have said the db doesn't exist: %w", err)
	}
}
