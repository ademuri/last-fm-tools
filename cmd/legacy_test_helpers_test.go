package cmd

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func createTestDb(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "lastfm.db")

	db, err := createDatabase(dbPath)
	if err != nil {
		t.Fatalf("createDatabase(%s) error: %v", dbPath, err)
	}
	if db == nil {
		t.Fatalf("createDatabase(%s) returned nil", dbPath)
	}

	return db, dbPath
}
