package cmd

import (
	"database/sql"
	"fmt"

	"github.com/ademuri/last-fm-tools/internal/migration"
)

// Legacy functions restored to support commands that haven't been refactored to use internal/store yet.

func createDatabase(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("createDatabase: %w", err)
	}
	err = createTables(database)
	if err != nil {
		return nil, fmt.Errorf("createDatabase: %w", err)
	}

	err = ensureSchema(database)
	if err != nil {
		return nil, fmt.Errorf("createDatabase: %w", err)
	}

	return database, nil
}

func ensureSchema(db *sql.DB) error {
	// Artist.tags_last_updated
	exists, err := columnExists(db, "Artist", "tags_last_updated")
	if err != nil {
		return fmt.Errorf("checking if Artist.tags_last_updated exists: %w", err)
	}
	if !exists {
		_, err := db.Exec("ALTER TABLE Artist ADD COLUMN tags_last_updated DATETIME")
		if err != nil {
			return fmt.Errorf("adding tags_last_updated to Artist: %w", err)
		}
	}

	// Album.tags_last_updated
	exists, err = columnExists(db, "Album", "tags_last_updated")
	if err != nil {
		return fmt.Errorf("checking if Album.tags_last_updated exists: %w", err)
	}
	if !exists {
		_, err := db.Exec("ALTER TABLE Album ADD COLUMN tags_last_updated DATETIME")
		if err != nil {
			return fmt.Errorf("adding tags_last_updated to Album: %w", err)
		}
	}

	// Report.params
	exists, err = columnExists(db, "Report", "params")
	if err != nil {
		return fmt.Errorf("checking if Report.params exists: %w", err)
	}
	if !exists {
		_, err := db.Exec("ALTER TABLE Report ADD COLUMN params TEXT")
		if err != nil {
			return fmt.Errorf("adding params to Report: %w", err)
		}
	}

	return nil
}

func createTables(db *sql.DB) error {
	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("createTables: %w", err)
	}

	if !exists {
		_, err = db.Exec(migration.Create)
		return err
	}

	return createTagTables(db)
}

func createTagTables(db *sql.DB) error {
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
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("createTagTables: %w", err)
	}
	return nil
}

func createUser(db *sql.DB, user string) error {
	userRows, err := db.Query("SELECT name FROM User WHERE name = ?", user)
	if err != nil {
		return fmt.Errorf("createUser(%q): %w", user, err)
	}
	defer userRows.Close()

	if !userRows.Next() {
		_, err := db.Exec("INSERT INTO User (name) VALUES (?)", user)
		if err != nil {
			return fmt.Errorf("createUser(%q): %w", user, err)
		}
	}
	return nil
}

func openDb(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("openDb: %w", err)
	}
	return db, nil
}

func dbExists(db *sql.DB) (bool, error) {
	exists, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'User'")
	if err != nil {
		return false, fmt.Errorf("createTables: %w", err)
	}
	defer exists.Close()

	return exists.Next(), nil
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
	return false, nil
}

func createArtist(db *sql.Tx, name string) (err error) {
	artistRows, err := db.Query("SELECT name FROM Artist WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("createArtist(%q): %w", name, err)
	}
	defer artistRows.Close()

	if !artistRows.Next() {
		_, err := db.Exec("INSERT INTO Artist (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("createArtist(%q): %w", name, err)
		}
	}

	return nil
}

func createAlbum(db *sql.Tx, artist string, name string) (err error) {
	albumRows, err := db.Query("SELECT name FROM Album WHERE artist = ? AND name = ?", artist, name)
	if err != nil {
		return fmt.Errorf("createAlbum(%q, %q): %w", artist, name, err)
	}
	defer albumRows.Close()

	if !albumRows.Next() {
		_, err := db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", artist, name)
		if err != nil {
			return fmt.Errorf("createAlbum(%q, %q): %w", artist, name, err)
		}
	}

	return nil
}

func createTrack(db *sql.Tx, artist string, album string, name string) (id int64, err error) {
	trackRows, err := db.Query("SELECT id FROM Track WHERE artist = ? AND album = ? AND name = ?", artist, album, name)
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}
	defer trackRows.Close()

	if trackRows.Next() {
		var id int64
		trackRows.Scan(&id)
		return id, nil
	}

	result, err := db.Exec("INSERT INTO Track (artist, album, name) VALUES (?, ?, ?)", artist, album, name)
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}

	track_id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}

	return track_id, nil
}

func createListen(db *sql.Tx, user string, track_id int64, datetime string) (err error) {
	listenRows, err := db.Query("SELECT id FROM Listen WHERE user = ? AND date = ? AND track = ?", user, datetime, track_id)
	if err != nil {
		return fmt.Errorf("createListen(%q, %q, %q): %w", user, track_id, datetime, err)
	}
	defer listenRows.Close()

	if listenRows.Next() {
		return nil
	}

	_, err = db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, ?, ?)", user, track_id, datetime)
	if err != nil {
		return fmt.Errorf("createListen(%q, %q, %q): %w", user, track_id, datetime, err)
	}
	return nil
}