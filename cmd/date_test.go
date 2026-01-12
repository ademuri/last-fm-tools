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

func TestGetImplicitDateRange_year(t *testing.T) {
	doTestGetImplicitDateRange(t, "2020", "2021", "2006")
}

func TestGetImplicitDateRange_month(t *testing.T) {
	doTestGetImplicitDateRange(t, "2020-01", "2020-02", "2006-01")
}

func TestGetImplicitDateRange_day(t *testing.T) {
	doTestGetImplicitDateRange(t, "2020-01-01", "2020-01-02", "2006-01-02")
}

func TestGetImplicitDateRange_invalid(t *testing.T) {
	tooMany := "2020-01-0123"
	_, _, err := getImplicitDateRange(tooMany)
	if err == nil {
		t.Fatalf("Expected error parsing %q", tooMany)
	}
	if !strings.Contains(err.Error(), "Invalid format") {
		t.Fatalf("Should have error with invalid format: %w", err)
	}

	letters := "not_real"
	_, _, err = getImplicitDateRange(letters)
	if err == nil {
		t.Fatalf("Expected error parsing %q", letters)
	}
	if !strings.Contains(err.Error(), "Invalid format") {
		t.Fatalf("Should have error with invalid format: %w", err)
	}
}

func doTestGetImplicitDateRange(t *testing.T, startString string, endString string, format string) {
	start, end, err := getImplicitDateRange(startString)
	if err != nil {
		t.Fatalf("Parsing year string: %w", err)
	}

	expectedStart, err := time.Parse(format, startString)
	if err != nil {
		t.Fatalf("Constructing expectedStart: %w", err)
	}

	expectedEnd, err := time.Parse(format, endString)
	if err != nil {
		t.Fatalf("Constructing expectedEnd: %w", err)
	}

	if start != expectedStart {
		t.Fatalf("Expected start to be %q, got %q", expectedStart, start)
	}

	if end != expectedEnd {
		t.Fatalf("Expected start to be %q, got %q", expectedEnd, end)
	}
}

func TestGetExplicitDateRange_valid(t *testing.T) {
	const startString = "2020"
	const endString = "2020-02-01"
	expectedStart, err := time.Parse("2006", startString)
	if err != nil {
		t.Fatalf("Constructing expectedStart: %w", err)
	}

	expectedEnd, err := time.Parse("2006-01-02", endString)
	if err != nil {
		t.Fatalf("Constructing expectedEnd: %w", err)
	}

	start, end, err := getExplicitDateRange(startString, endString)
	if err != nil {
		t.Fatalf("getExplicitDateRange(%q, %q): %w", startString, endString, err)
	}

	if start != expectedStart {
		t.Fatalf("Expected start to be %q, got %q", expectedStart, start)
	}

	if end != expectedEnd {
		t.Fatalf("Expected start to be %q, got %q", expectedEnd, end)
	}
}

func TestParseSingleDatestring_Relative(t *testing.T) {
	tests := []struct {
		input    string
		unit     string
		amount   int
	}{
		{"30d", "d", 30},
		{"12w", "w", 12},
		{"6m", "m", 6},
		{"10y", "y", 10},
	}

	for _, tc := range tests {
		pd, err := parseSingleDatestring(tc.input)
		if err != nil {
			t.Errorf("parseSingleDatestring(%q) returned error: %v", tc.input, err)
			continue
		}

		// Calculate expected approximate time
		now := time.Now()
		var expected time.Time
		switch tc.unit {
		case "d":
			expected = now.AddDate(0, 0, -tc.amount)
		case "w":
			expected = now.AddDate(0, 0, -tc.amount*7)
		case "m":
			expected = now.AddDate(0, -tc.amount, 0)
		case "y":
			expected = now.AddDate(-tc.amount, 0, 0)
		}

		// Check if result is close to expected (within 1 second)
		diff := pd.Date.Sub(expected)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("parseSingleDatestring(%q) = %v; want approx %v", tc.input, pd.Date, expected)
		}
	}
}

func TestGetExplicitDateRange_invalid(t *testing.T) {
	_, _, err := getExplicitDateRange("2020", "abc")
	if err == nil {
		t.Fatalf("Expected error when parsing invalid datestring")
	}
}
