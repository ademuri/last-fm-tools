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
	"testing"
	"time"
)

func TestGetNumYearsOfListeningData(t *testing.T) {
	db, err := createTestDb()
	if err != nil {
		t.Fatalf("createTestDb() error: %w", err)
	}

	user := "testuser"
	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %w", user, err)
	}

	now := time.Now()
	err = createListenForDate(db, user, now)
	if err != nil {
		t.Fatalf("createListenForDate: %w", err)
	}

	years, err := getNumYearsOfListeningData(db, user)
	if err != nil {
		t.Fatalf("getNumYearsOfListeningData(): %w", err)
	}
	if years != 1 {
		t.Errorf("Expected 1 year, got %d", years)
	}

	oneYearAgo := time.Date(now.Year()-1, now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	err = createListenForDate(db, user, oneYearAgo)
	if err != nil {
		t.Fatalf("createListenForDate: %w", err)
	}
	years, err = getNumYearsOfListeningData(db, user)
	if err != nil {
		t.Fatalf("getNumYearsOfListeningData(): %w", err)
	}
	if years != 1 {
		t.Errorf("Expected 1 year, got %d", years)
	}

	oneYearMinusOneDayAgo := time.Date(now.Year()-1, now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err = createListenForDate(db, user, oneYearMinusOneDayAgo)
	if err != nil {
		t.Fatalf("createListenForDate: %w", err)
	}
	years, err = getNumYearsOfListeningData(db, user)
	if err != nil {
		t.Fatalf("getNumYearsOfListeningData(): %w", err)
	}
	if years != 2 {
		t.Errorf("Expected 2 year, got %d", years)
	}

	threeYearsOneMonthAgo := time.Date(now.Year()-3, now.Month()-1, now.Day(), 0, 0, 0, 0, now.Location())
	err = createListenForDate(db, user, threeYearsOneMonthAgo)
	if err != nil {
		t.Fatalf("createListenForDate: %w", err)
	}
	years, err = getNumYearsOfListeningData(db, user)
	if err != nil {
		t.Fatalf("getNumYearsOfListeningData(): %w", err)
	}
	if years != 4 {
		t.Errorf("Expected 4 year, got %d", years)
	}
}

func createListenForDate(db *sql.DB, user string, time time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("creating transaction: %w", err)
	}
	err = createListen(tx, user, 1, strconv.FormatInt(time.Unix(), 10))
	if err != nil {
		return fmt.Errorf("createListen(): %w", err)
	}
	tx.Commit()

	return nil
}
