package analysis

import (
	"fmt"
	"sort"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
)

type ForgottenConfig struct {
	LastListenAfter    time.Time
	LastListenBefore   time.Time
	FirstListenAfter   time.Time
	FirstListenBefore  time.Time
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

func determineBand(scrobbles int64, isArtist bool) string {
	if isArtist {
		if scrobbles >= ThresholdArtistObsession {
			return BandObsession
		}
		if scrobbles >= ThresholdArtistStrong {
			return BandStrong
		}
		if scrobbles >= ThresholdArtistModerate {
			return BandModerate
		}
	} else {
		if scrobbles >= ThresholdAlbumObsession {
			return BandObsession
		}
		if scrobbles >= ThresholdAlbumStrong {
			return BandStrong
		}
		if scrobbles >= ThresholdAlbumModerate {
			return BandModerate
		}
	}
	return ""
}

func GetForgottenArtists(db *store.Store, user string, cfg ForgottenConfig, now time.Time) (map[string][]ForgottenArtist, error) {
	opts := store.ForgottenQueryOptions{
		MinScrobbles:      cfg.MinArtistScrobbles,
		LastListenAfter:   cfg.LastListenAfter.Unix(),
		LastListenBefore:  cfg.LastListenBefore.Unix(),
		FirstListenAfter:  cfg.FirstListenAfter.Unix(),
		FirstListenBefore: cfg.FirstListenBefore.Unix(),
	}

	stats, err := db.GetForgottenArtists(user, opts)
	if err != nil {
		return nil, fmt.Errorf("getting forgotten artists: %w", err)
	}

	results := make(map[string][]ForgottenArtist)

	for _, s := range stats {
		a := ForgottenArtist{
			Artist:         s.Artist,
			TotalScrobbles: s.TotalScrobbles,
			FirstListen:    s.FirstListen,
			LastListen:     s.LastListen,
			DaysSinceLast:  int(now.Sub(s.LastListen).Hours() / 24),
		}

		a.Band = determineBand(a.TotalScrobbles, true)
		if a.Band == "" {
			continue
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

func GetForgottenAlbums(db *store.Store, user string, cfg ForgottenConfig, now time.Time) (map[string][]ForgottenAlbum, error) {
	opts := store.ForgottenQueryOptions{
		MinScrobbles:      cfg.MinAlbumScrobbles,
		LastListenAfter:   cfg.LastListenAfter.Unix(),
		LastListenBefore:  cfg.LastListenBefore.Unix(),
		FirstListenAfter:  cfg.FirstListenAfter.Unix(),
		FirstListenBefore: cfg.FirstListenBefore.Unix(),
	}

	stats, err := db.GetForgottenAlbums(user, opts)
	if err != nil {
		return nil, fmt.Errorf("getting forgotten albums: %w", err)
	}

	results := make(map[string][]ForgottenAlbum)

	for _, s := range stats {
		a := ForgottenAlbum{
			Artist:         s.Artist,
			Album:          s.Album,
			TotalScrobbles: s.TotalScrobbles,
			FirstListen:    s.FirstListen,
			LastListen:     s.LastListen,
			DaysSinceLast:  int(now.Sub(s.LastListen).Hours() / 24),
		}

		a.Band = determineBand(a.TotalScrobbles, false)
		if a.Band == "" {
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
		// Default to dormancy (longest dormancy first)
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