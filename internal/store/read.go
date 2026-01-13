package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func (s *Store) GetSessionKey(user string) (string, error) {
	row := s.db.QueryRow("SELECT session_key FROM User WHERE name = ? AND session_key <> ''", user)
	var key string
	err := row.Scan(&key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting session key: %w", err)
	}
	return key, nil
}

func (s *Store) GetLastUpdated(user string) (time.Time, error) {
	row := s.db.QueryRow("SELECT last_updated FROM User WHERE name = ?", user)
	var t sql.NullTime
	err := row.Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("getting last updated: %w", err)
	}
	return t.Time, nil
}

func (s *Store) GetLatestListen(user string) (time.Time, error) {
	// Matches legacy logic: sort by CAST(date AS INTEGER)
	query := "SELECT date FROM Listen WHERE user = ? ORDER BY CAST(date AS INTEGER) desc LIMIT 1"
	row := s.db.QueryRow(query, user)
	var dateStr string
	err := row.Scan(&dateStr)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("scanning latest listen: %w", err)
	}

	return parseDate(dateStr)
}

func parseDate(dateStr string) (time.Time, error) {
	// Legacy logic handles both Unix timestamp (as string) and ISO8601
	dateInt, err := strconv.ParseInt(dateStr, 10, 64)
	if err == nil {
		return time.Unix(dateInt, 0), nil
	}
	
	t, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}
	
	return time.Time{}, fmt.Errorf("parsing date %q: %w", dateStr, err)
}

// Tag Update Helpers

func (s *Store) GetArtistsNeedingTagUpdate(interval time.Duration) ([]string, error) {
	threshold := time.Now().Add(-interval)
	query := `
		SELECT t.artist
		FROM Listen l
		JOIN Track t ON l.track = t.id
		JOIN Artist a ON t.artist = a.name
		WHERE (a.tags_last_updated IS NULL OR a.tags_last_updated < ?)
		GROUP BY t.artist
		HAVING COUNT(*) > 10
	`
	rows, err := s.db.Query(query, threshold)
	if err != nil {
		return nil, fmt.Errorf("querying artists for tag update: %w", err)
	}
	defer rows.Close()

	var artists []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

type AlbumKey struct {
	Artist string
	Name   string
}

func (s *Store) GetAlbumsNeedingTagUpdate(interval time.Duration) ([]AlbumKey, error) {
	threshold := time.Now().Add(-interval)
	query := `
		SELECT t.artist, t.album
		FROM Listen l
		JOIN Track t ON l.track = t.id
		JOIN Album a ON t.artist = a.artist AND t.album = a.name
		WHERE t.album != "" AND (a.tags_last_updated IS NULL OR a.tags_last_updated < ?)
		GROUP BY t.artist, t.album
		HAVING COUNT(*) > 10
	`
	rows, err := s.db.Query(query, threshold)
	if err != nil {
		return nil, fmt.Errorf("querying albums for tag update: %w", err)
	}
	defer rows.Close()

	var albums []AlbumKey
	for rows.Next() {
		var k AlbumKey
		if err := rows.Scan(&k.Artist, &k.Name); err != nil {
			return nil, err
		}
		albums = append(albums, k)
	}
	return albums, rows.Err()
}

func (s *Store) GetListensInRange(user string, start, end time.Time) ([]time.Time, error) {
	startUTS := start.Unix()
	endUTS := end.Unix()

	query := `
		SELECT date 
		FROM Listen 
		WHERE user = ? 
		AND CAST(date AS INTEGER) >= ? 
		AND CAST(date AS INTEGER) < ?
		ORDER BY CAST(date AS INTEGER) ASC
	`

	rows, err := s.db.Query(query, user, startUTS, endUTS)
	if err != nil {
		return nil, fmt.Errorf("querying listens: %w", err)
	}
	defer rows.Close()

	var listens []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return nil, err
		}
		t, err := parseDate(dateStr)
		if err != nil {
			continue
		}
		listens = append(listens, t)
	}
	return listens, rows.Err()
}
