package store

import (
	"database/sql"
	"fmt"
	"time"
)

type TrackImport struct {
	Artist    string
	Album     string
	TrackName string
	DateUTS   string // Keep as string to match legacy input, or parse before?
}

// CreateUser ensures a user exists in the database.
func (s *Store) CreateUser(user string) error {
	row := s.db.QueryRow("SELECT name FROM User WHERE name = ?", user)
	var name string
	err := row.Scan(&name)
	if err == sql.ErrNoRows {
		_, err := s.db.Exec("INSERT INTO User (name) VALUES (?)", user)
		if err != nil {
			return fmt.Errorf("inserting user %q: %w", user, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking user %q: %w", user, err)
	}
	return nil
}

func (s *Store) SetLastUpdated(user string, updated time.Time) error {
	_, err := s.db.Exec("UPDATE User SET last_updated = ? WHERE name = ?", updated, user)
	if err != nil {
		return fmt.Errorf("updating last_updated for %q: %w", user, err)
	}
	return nil
}

// AddRecentTracks inserts a batch of tracks transactionally.
func (s *Store) AddRecentTracks(user string, tracks []TrackImport) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for _, track := range tracks {
		if err := createArtist(tx, track.Artist); err != nil {
			return err
		}
		if err := createAlbum(tx, track.Artist, track.Album); err != nil {
			return err
		}
		trackID, err := createTrack(tx, track.Artist, track.Album, track.TrackName)
		if err != nil {
			return err
		}
		if err := createListen(tx, user, trackID, track.DateUTS); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// Internal helper functions (private, taking *sql.Tx)

func createArtist(tx *sql.Tx, name string) error {
	// Optimization: we could cache artists in a map during the transaction if needed,
	// but for now relying on DB uniqueness checks.
	// Using INSERT OR IGNORE or checking existence first.
	// Legacy code checked existence.
	var dummy string
	err := tx.QueryRow("SELECT name FROM Artist WHERE name = ?", name).Scan(&dummy)
	if err == sql.ErrNoRows {
		_, err := tx.Exec("INSERT INTO Artist (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("inserting artist %q: %w", name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking artist %q: %w", name, err)
	}
	return nil
}

func createAlbum(tx *sql.Tx, artist, name string) error {
	var dummy string
	err := tx.QueryRow("SELECT name FROM Album WHERE artist = ? AND name = ?", artist, name).Scan(&dummy)
	if err == sql.ErrNoRows {
		_, err := tx.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", artist, name)
		if err != nil {
			return fmt.Errorf("inserting album %q for %q: %w", name, artist, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking album %q: %w", name, err)
	}
	return nil
}

func createTrack(tx *sql.Tx, artist, album, name string) (int64, error) {
	var id int64
	err := tx.QueryRow("SELECT id FROM Track WHERE artist = ? AND album = ? AND name = ?", artist, album, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("checking track %q: %w", name, err)
	}

	res, err := tx.Exec("INSERT INTO Track (artist, album, name) VALUES (?, ?, ?)", artist, album, name)
	if err != nil {
		return 0, fmt.Errorf("inserting track %q: %w", name, err)
	}
	return res.LastInsertId()
}

func createListen(tx *sql.Tx, user string, trackID int64, date string) error {
	// Check for duplicate listen
	var dummy int64
	err := tx.QueryRow("SELECT id FROM Listen WHERE user = ? AND date = ? AND track = ?", user, date, trackID).Scan(&dummy)
	if err == nil {
		return nil // Already exists
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("checking listen: %w", err)
	}

	_, err = tx.Exec("INSERT INTO Listen (user, track, date) VALUES (?, ?, ?)", user, trackID, date)
	if err != nil {
		return fmt.Errorf("inserting listen: %w", err)
	}
	return nil
}

// Tag Operations

func (s *Store) SaveArtistTags(artist string, tags []string, counts []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, tag := range tags {
		count := 0
		if i < len(counts) {
			count = counts[i]
		}

		// Ensure Tag exists
		_, err := tx.Exec("INSERT OR IGNORE INTO Tag (name) VALUES (?)", tag)
		if err != nil {
			return fmt.Errorf("inserting tag %q: %w", tag, err)
		}

		// Insert/Update ArtistTag
		_, err = tx.Exec("INSERT OR REPLACE INTO ArtistTag (artist, tag, count) VALUES (?, ?, ?)", artist, tag, count)
		if err != nil {
			return fmt.Errorf("linking tag %q to artist %q: %w", tag, artist, err)
		}
	}

	if err := s.MarkArtistTagsUpdated(tx, artist); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) MarkArtistTagsUpdated(tx *sql.Tx, artist string) error {
	query := "UPDATE Artist SET tags_last_updated = ? WHERE name = ?"
	
	var err error
	if tx != nil {
		_, err = tx.Exec(query, time.Now(), artist)
	} else {
		_, err = s.db.Exec(query, time.Now(), artist)
	}
	
	if err != nil {
		return fmt.Errorf("updating artist tag timestamp: %w", err)
	}
	return nil
}

func (s *Store) SaveAlbumTags(artist, album string, tags []string, counts []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, tag := range tags {
		count := 0
		if i < len(counts) {
			count = counts[i]
		}

		_, err := tx.Exec("INSERT OR IGNORE INTO Tag (name) VALUES (?)", tag)
		if err != nil {
			return fmt.Errorf("inserting tag %q: %w", tag, err)
		}

		_, err = tx.Exec("INSERT OR REPLACE INTO AlbumTag (artist, album, tag, count) VALUES (?, ?, ?, ?)", artist, album, tag, count)
		if err != nil {
			return fmt.Errorf("linking tag %q to album %q: %w", tag, album, err)
		}
	}

	if err := s.MarkAlbumTagsUpdated(tx, artist, album); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) MarkAlbumTagsUpdated(tx *sql.Tx, artist, album string) error {
	query := "UPDATE Album SET tags_last_updated = ? WHERE artist = ? AND name = ?"
	
	var err error
	if tx != nil {
		_, err = tx.Exec(query, time.Now(), artist, album)
	} else {
		_, err = s.db.Exec(query, time.Now(), artist, album)
	}
	
	if err != nil {
		return fmt.Errorf("updating album tag timestamp: %w", err)
	}
	return nil
}
