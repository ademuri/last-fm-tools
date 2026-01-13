package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ademuri/last-fm-tools/internal/migration"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensuring schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func createTables(db *sql.DB) error {
	// Check if main tables exist
	exists, err := dbExists(db)
	if err != nil {
		return err
	}

	if !exists {
		if _, err := db.Exec(migration.Create); err != nil {
			return fmt.Errorf("executing migration: %w", err)
		}
	}

	return createTagTables(db)
}

func dbExists(db *sql.DB) (bool, error) {
	// Check for 'User' table as a proxy for DB existence
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'User'")
	var name string
	err := row.Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking db existence: %w", err)
	}
	return true, nil
}

func createTagTables(db *sql.DB) error {
	// Tag tables are in migration.Create now (checked file), but cmd/update.go had a separate function.
	// Let's check if migration.Create includes them.
	// Yes, I read create-tables.sql and it includes Tag, ArtistTag, AlbumTag.
	// So we might not need this if migration.Create is always run for new DBs.
	// BUT, if migration.Create was run in the past without them, we need to add them.
	// Since migration.Create contains them now, any *new* DB will have them.
	// For *existing* DBs, we might need a migration step.
	// cmd/update.go unconditionally ran createTagTables.
	// The safest bet is to run the IF NOT EXISTS queries again or trust the migration logic.
	// Let's rely on ensureSchema or similar for updates, but for now I'll include the conditional creation here
	// to match legacy behavior, but cleaner.
	
	query := `
CREATE TABLE IF NOT EXISTS Tag (
  name TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS ArtistTag (
  artist TEXT,
  tag TEXT,
  count INTEGER,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  FOREIGN KEY (tag) REFERENCES Tag(name),
  PRIMARY KEY (artist, tag)
);

CREATE TABLE IF NOT EXISTS AlbumTag (
  artist TEXT,
  album TEXT,
  tag TEXT,
  count INTEGER,
  FOREIGN KEY (artist) REFERENCES Artist(name),
  FOREIGN KEY (album) REFERENCES Album(name),
  FOREIGN KEY (tag) REFERENCES Tag(name),
  PRIMARY KEY (artist, album, tag)
);
`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("creating tag tables: %w", err)
	}
	return nil
}

func ensureSchema(db *sql.DB) error {
	// Artist.tags_last_updated
	if err := addColumnIfNotExists(db, "Artist", "tags_last_updated", "DATETIME"); err != nil {
		return err
	}
	// Album.tags_last_updated
	if err := addColumnIfNotExists(db, "Album", "tags_last_updated", "DATETIME"); err != nil {
		return err
	}

	// CommandHistory table
	query := `CREATE TABLE CommandHistory (
  command_name TEXT,
  user TEXT,
  last_run DATETIME,
  PRIMARY KEY (command_name, user)
)`
	if err := createTableIfNotExists(db, "CommandHistory", query); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetCommandLastRun(user, command string) (time.Time, error) {
	row := s.db.QueryRow("SELECT last_run FROM CommandHistory WHERE user = ? AND command_name = ?", user, command)
	var lastRun time.Time
	err := row.Scan(&lastRun)
	if err == sql.ErrNoRows {
		return time.Time{}, nil // Never run before
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("getting last run: %w", err)
	}
	return lastRun, nil
}

func (s *Store) SetCommandLastRun(user, command string, t time.Time) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO CommandHistory (user, command_name, last_run) VALUES (?, ?, ?)`, user, command, t)
	if err != nil {
		return fmt.Errorf("setting last run: %w", err)
	}
	return nil
}

func createTableIfNotExists(db *sql.DB, tableName, schema string) error {
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name = ?", tableName)
	var name string
	err := row.Scan(&name)
	if err == sql.ErrNoRows {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("creating table %s: %w", tableName, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking table existence %s: %w", tableName, err)
	}
	return nil
}

func addColumnIfNotExists(db *sql.DB, table, column, typeDef string) error {
	exists, err := columnExists(db, table, column)
	if err != nil {
		return fmt.Errorf("checking column %s.%s: %w", table, column, err)
	}
	if !exists {
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, typeDef)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("adding column %s.%s: %w", table, column, err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, tableName string, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt_value interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err() // Added error check
}
