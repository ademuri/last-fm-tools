package store

import (
	"fmt"
	"time"
)

type ArtistPlayCount struct {
	Artist string
	Count  int64
}

type AlbumPlayCount struct {
	Artist string
	Album  string
	Count  int64
}

func (s *Store) GetTopArtistsWithCount(user string, start, end time.Time) ([]ArtistPlayCount, error) {
	query := `
	SELECT Track.artist, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist
	ORDER BY COUNT(*) DESC
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("querying top artists: %w", err)
	}
	defer rows.Close()

	var results []ArtistPlayCount
	for rows.Next() {
		var apc ArtistPlayCount
		if err := rows.Scan(&apc.Artist, &apc.Count); err != nil {
			return nil, err
		}
		results = append(results, apc)
	}
	return results, rows.Err()
}

func (s *Store) GetTopAlbumsWithCount(user string, start, end time.Time) ([]AlbumPlayCount, error) {
	query := `
	SELECT Track.artist, Track.album, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist, Track.album
	ORDER BY COUNT(*) DESC
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("querying top albums: %w", err)
	}
	defer rows.Close()

	var results []AlbumPlayCount
	for rows.Next() {
		var apc AlbumPlayCount
		if err := rows.Scan(&apc.Artist, &apc.Album, &apc.Count); err != nil {
			return nil, err
		}
		results = append(results, apc)
	}
	return results, rows.Err()
}
