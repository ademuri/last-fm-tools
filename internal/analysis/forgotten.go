package analysis

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type ForgottenConfig struct {
	DormancyDays       int
	MinArtistScrobbles int
	MinAlbumScrobbles  int
	ResultsPerBand     int
	SortBy             string // "dormancy" or "listens"
}

type ForgottenArtist struct {
	Artist         string
	TotalScrobbles int64
	FirstListen    time.Time
	LastListen     time.Time
	DaysSinceLast  int
	Band           string
}

type ForgottenAlbum struct {
	Artist         string
	Album          string
	TotalScrobbles int64
	FirstListen    time.Time
	LastListen     time.Time
	DaysSinceLast  int
	Band           string
}

const (
	BandObsession = "Obsession"
	BandStrong    = "Strong"
	BandModerate  = "Moderate"

	// Artist Thresholds
	ThresholdArtistObsession = 120
	ThresholdArtistStrong    = 50
	ThresholdArtistModerate  = 15

	// Album Thresholds
	ThresholdAlbumObsession = 60
	ThresholdAlbumStrong    = 30
	ThresholdAlbumModerate  = 10
)

// GetThreshold returns the minimum scrobbles for a given band and type (artist/album).
func GetThreshold(band string, isArtist bool) int {
	if isArtist {
		switch band {
		case BandObsession:
			return ThresholdArtistObsession
		case BandStrong:
			return ThresholdArtistStrong
		case BandModerate:
			return ThresholdArtistModerate
		}
	} else {
		switch band {
		case BandObsession:
			return ThresholdAlbumObsession
		case BandStrong:
			return ThresholdAlbumStrong
		case BandModerate:
			return ThresholdAlbumModerate
		}
	}
	return 0
}

func GetForgottenArtists(db *sql.DB, user string, cfg ForgottenConfig) (map[string][]ForgottenArtist, error) {
	dormancyLimit := time.Now().AddDate(0, 0, -cfg.DormancyDays)

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
		HAVING total_scrobbles >= ? AND last_listen < ?
	`

	rows, err := db.Query(query, user, cfg.MinArtistScrobbles, dormancyLimit.Unix())
	if err != nil {
		return nil, fmt.Errorf("querying forgotten artists: %w", err)
	}
	defer rows.Close()

	results := make(map[string][]ForgottenArtist)
	now := time.Now()

	for rows.Next() {
		var a ForgottenArtist
		var first, last int64
		if err := rows.Scan(&a.Artist, &a.TotalScrobbles, &first, &last); err != nil {
			return nil, err
		}
		a.FirstListen = time.Unix(first, 0)
		a.LastListen = time.Unix(last, 0)
		a.DaysSinceLast = int(now.Sub(a.LastListen).Hours() / 24)

		if a.TotalScrobbles >= ThresholdArtistObsession {
			a.Band = BandObsession
		} else if a.TotalScrobbles >= ThresholdArtistStrong {
			a.Band = BandStrong
		} else if a.TotalScrobbles >= ThresholdArtistModerate {
			a.Band = BandModerate
		} else {
			continue // Should be filtered by SQL, but good safety
		}

		results[a.Band] = append(results[a.Band], a)
	}

	for band := range results {
		sortArtists(results[band], cfg.SortBy)
		if len(results[band]) > cfg.ResultsPerBand {
			results[band] = results[band][:cfg.ResultsPerBand]
		}
	}

	return results, nil
}

func GetForgottenAlbums(db *sql.DB, user string, cfg ForgottenConfig) (map[string][]ForgottenAlbum, error) {
	dormancyLimit := time.Now().AddDate(0, 0, -cfg.DormancyDays)

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
		HAVING total_scrobbles >= ? AND last_listen < ?
	`

	rows, err := db.Query(query, user, cfg.MinAlbumScrobbles, dormancyLimit.Unix())
	if err != nil {
		return nil, fmt.Errorf("querying forgotten albums: %w", err)
	}
	defer rows.Close()

	results := make(map[string][]ForgottenAlbum)
	now := time.Now()

	for rows.Next() {
		var a ForgottenAlbum
		var first, last int64
		if err := rows.Scan(&a.Artist, &a.Album, &a.TotalScrobbles, &first, &last); err != nil {
			return nil, err
		}
		a.FirstListen = time.Unix(first, 0)
		a.LastListen = time.Unix(last, 0)
		a.DaysSinceLast = int(now.Sub(a.LastListen).Hours() / 24)

		if a.TotalScrobbles >= ThresholdAlbumObsession {
			a.Band = BandObsession
		} else if a.TotalScrobbles >= ThresholdAlbumStrong {
			a.Band = BandStrong
		} else if a.TotalScrobbles >= ThresholdAlbumModerate {
			a.Band = BandModerate
		} else {
			continue
		}

		results[a.Band] = append(results[a.Band], a)
	}

	for band := range results {
		sortAlbums(results[band], cfg.SortBy)
		if len(results[band]) > cfg.ResultsPerBand {
			results[band] = results[band][:cfg.ResultsPerBand]
		}
	}

	return results, nil
}

func sortArtists(artists []ForgottenArtist, sortBy string) {
	sort.Slice(artists, func(i, j int) bool {
		if sortBy == "listens" {
			return artists[i].TotalScrobbles > artists[j].TotalScrobbles
		}
		// Default to dormancy (longest dormancy first? or shortest?)
		// "The goal is to help rediscover music" -> probably want the ones I haven't heard in longest time?
		// Or maybe the ones that just slipped out (shortest dormancy > threshold)?
		// Usually "Forgotten" implies "Long time ago". Let's sort by DaysSinceLast DESC.
		return artists[i].DaysSinceLast > artists[j].DaysSinceLast
	})
}

func sortAlbums(albums []ForgottenAlbum, sortBy string) {
	sort.Slice(albums, func(i, j int) bool {
		if sortBy == "listens" {
			return albums[i].TotalScrobbles > albums[j].TotalScrobbles
		}
		return albums[i].DaysSinceLast > albums[j].DaysSinceLast
	})
}
