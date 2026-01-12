package store

import (
	"fmt"
	"time"
)

type ForgottenQueryOptions struct {
	MinScrobbles      int
	LastListenAfter   int64
	LastListenBefore  int64
	FirstListenAfter  int64
	FirstListenBefore int64
}

type ArtistListenStats struct {
	Artist         string
	TotalScrobbles int64
	FirstListen    time.Time
	LastListen     time.Time
}

type AlbumListenStats struct {
	Artist         string
	Album          string
	TotalScrobbles int64
	FirstListen    time.Time
	LastListen     time.Time
}

func (s *Store) GetForgottenArtists(user string, opts ForgottenQueryOptions) ([]ArtistListenStats, error) {
	query := `
		SELECT
			t.artist,
			COUNT(*) as total_scrobbles,
			MIN(l.date) as first_listen,
			MAX(l.date) as last_listen
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ?
		GROUP BY t.artist
		HAVING total_scrobbles >= ? AND last_listen >= ? AND last_listen <= ? AND first_listen >= ? AND first_listen <= ?
	`

	rows, err := s.db.Query(query, user, opts.MinScrobbles, opts.LastListenAfter, opts.LastListenBefore, opts.FirstListenAfter, opts.FirstListenBefore)
	if err != nil {
		return nil, fmt.Errorf("querying forgotten artists: %w", err)
	}
	defer rows.Close()

	var stats []ArtistListenStats
	for rows.Next() {
		var a ArtistListenStats
		var first, last int64
		if err := rows.Scan(&a.Artist, &a.TotalScrobbles, &first, &last); err != nil {
			return nil, err
		}
		a.FirstListen = time.Unix(first, 0)
		a.LastListen = time.Unix(last, 0)
		stats = append(stats, a)
	}
	return stats, rows.Err()
}

func (s *Store) GetForgottenAlbums(user string, opts ForgottenQueryOptions) ([]AlbumListenStats, error) {
	query := `
		SELECT
			t.artist,
			t.album,
			COUNT(*) as total_scrobbles,
			MIN(l.date) as first_listen,
			MAX(l.date) as last_listen
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.album != ''
		GROUP BY t.artist, t.album
		HAVING total_scrobbles >= ? AND last_listen >= ? AND last_listen <= ? AND first_listen >= ? AND first_listen <= ?
	`

	rows, err := s.db.Query(query, user, opts.MinScrobbles, opts.LastListenAfter, opts.LastListenBefore, opts.FirstListenAfter, opts.FirstListenBefore)
	if err != nil {
		return nil, fmt.Errorf("querying forgotten albums: %w", err)
	}
	defer rows.Close()

	var stats []AlbumListenStats
	for rows.Next() {
		var a AlbumListenStats
		var first, last int64
		if err := rows.Scan(&a.Artist, &a.Album, &a.TotalScrobbles, &first, &last); err != nil {
			return nil, err
		}
		a.FirstListen = time.Unix(first, 0)
		a.LastListen = time.Unix(last, 0)
		stats = append(stats, a)
	}
	return stats, rows.Err()
}
