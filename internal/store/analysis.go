package store

import (
	"fmt"
	"time"
)

type ArtistScrobbleCount struct {
	Name      string
	Scrobbles int64
}

type AlbumScrobbleCount struct {
	Title     string
	Artist    string
	Scrobbles int64
}

type TagCount struct {
	Tag   string
	Count int
}

func (s *Store) GetTotalScrobbles(user string) (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user).Scan(&count)
	return count, err
}

func (s *Store) GetTotalArtists(user string) (int, error) {
	var count int
	query := `SELECT COUNT(DISTINCT t.artist) FROM Listen l JOIN Track t ON l.track = t.id WHERE l.user = ?`
	err := s.db.QueryRow(query, user).Scan(&count)
	return count, err
}

func (s *Store) GetFirstListen(user string) (time.Time, error) {
	var date int64
	err := s.db.QueryRow("SELECT MIN(date) FROM Listen WHERE user = ?", user).Scan(&date)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(date, 0), nil
}

func (s *Store) GetTopArtists(user string, start, end time.Time, limit int) ([]ArtistScrobbleCount, error) {
	query := `
		SELECT t.artist, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ?
		GROUP BY t.artist
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, fmt.Errorf("querying top artists: %w", err)
	}
	defer rows.Close()

	var artists []ArtistScrobbleCount
	for rows.Next() {
		var a ArtistScrobbleCount
		if err := rows.Scan(&a.Name, &a.Scrobbles); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

func (s *Store) GetTopAlbumsForArtist(user, artist string, start, end time.Time, limit int) ([]TagCount, error) {
	// Reusing TagCount struct for Name/Count pair
	query := `
		SELECT t.album, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.album
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, user, artist, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []TagCount
	for rows.Next() {
		var a TagCount
		if err := rows.Scan(&a.Tag, &a.Count); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

func (s *Store) GetTopTagsForArtist(artist string, limit int) ([]string, error) {
	query := `SELECT tag FROM ArtistTag WHERE artist = ? ORDER BY count DESC LIMIT ?`
	rows, err := s.db.Query(query, artist, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) GetTopAlbums(user string, start, end time.Time, limit int) ([]AlbumScrobbleCount, error) {
	query := `
		SELECT t.album, t.artist, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.album, t.artist
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []AlbumScrobbleCount
	for rows.Next() {
		var a AlbumScrobbleCount
		if err := rows.Scan(&a.Title, &a.Artist, &a.Scrobbles); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

func (s *Store) GetTopTagsForAlbum(artist, album string, limit int) ([]string, error) {
	query := `SELECT tag FROM AlbumTag WHERE artist = ? AND album = ? ORDER BY count DESC LIMIT ?`
	rows, err := s.db.Query(query, artist, album, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) GetArtistListenCount(user, artist string, start, end time.Time) (int64, error) {
	query := `
		SELECT COUNT(*) 
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ? AND l.date BETWEEN ? AND ?
	`
	var count int64
	err := s.db.QueryRow(query, user, artist, start.Unix(), end.Unix()).Scan(&count)
	return count, err
}

// GetPeakYears returns the start and end year of the peak listening period.
// Returns "year" or "start-end".
func (s *Store) GetPeakYears(user, artist string) (string, error) {
	query := `
		SELECT strftime('%Y', datetime(date, 'unixepoch')) as year, COUNT(*)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ?
		GROUP BY year
		ORDER BY year
	`
	rows, err := s.db.Query(query, user, artist)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type yearCount struct {
		year  string
		count int
	}
	var counts []yearCount
	var total int
	for rows.Next() {
		var yc yearCount
		if err := rows.Scan(&yc.year, &yc.count); err != nil {
			return "", err
		}
		counts = append(counts, yc)
		total += yc.count
	}

	if total == 0 {
		return "", nil
	}

	// Logic moved from analysis.go to here (or keep it there? It's logic, not just data access).
	// But it requires iterating the result set. 
	// I'll keep the calculation logic here to encapsulate the raw SQL/iteration.
	
	target := int(float64(total) * 0.8)
	
	bestStart, bestEnd := -1, -1
	minLen := 999999

	for i := 0; i < len(counts); i++ {
		currentSum := 0
		for j := i; j < len(counts); j++ {
			currentSum += counts[j].count
			if currentSum >= target {
				length := j - i + 1
				if length < minLen {
					minLen = length
					bestStart = i
					bestEnd = j
				}
				break
			}
		}
	}

	if bestStart != -1 {
		if bestStart == bestEnd {
			return counts[bestStart].year, nil
		}
		return fmt.Sprintf("%s-%s", counts[bestStart].year, counts[bestEnd].year), nil
	}

	return "Unknown", nil
}

func (s *Store) GetAverageTracksPerAlbum(user string, start, end time.Time) (float64, error) {
	query := `
		SELECT COUNT(DISTINCT t.id)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.artist, t.album
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var countSum int64
	var albumCount int64
	for rows.Next() {
		var c int64
		if err := rows.Scan(&c); err != nil {
			return 0, err
		}
		countSum += c
		albumCount++
	}

	if albumCount == 0 {
		return 0, nil
	}
	return float64(countSum) / float64(albumCount), nil
}

// Support for Tag Weighting

type ArtistTagData struct {
	Artist string
	Tag    string
	Count  int
}

type AlbumTagData struct {
	Artist string
	Album  string
	Tag    string
	Count  int
}

func (s *Store) GetAllArtistTags() ([]ArtistTagData, error) {
	rows, err := s.db.Query("SELECT artist, tag, count FROM ArtistTag")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []ArtistTagData
	for rows.Next() {
		var d ArtistTagData
		if err := rows.Scan(&d.Artist, &d.Tag, &d.Count); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (s *Store) GetAllAlbumTags() ([]AlbumTagData, error) {
	rows, err := s.db.Query("SELECT artist, album, tag, count FROM AlbumTag ORDER BY artist, album")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []AlbumTagData
	for rows.Next() {
		var d AlbumTagData
		if err := rows.Scan(&d.Artist, &d.Album, &d.Tag, &d.Count); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (s *Store) GetAlbumListenCounts(user string, start, end time.Time) ([]AlbumScrobbleCount, error) {
	// Reusing AlbumScrobbleCount
	query := `
		SELECT t.artist, t.album, COUNT(*)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ?
		GROUP BY t.artist, t.album
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []AlbumScrobbleCount
	for rows.Next() {
		var c AlbumScrobbleCount
		if err := rows.Scan(&c.Artist, &c.Title, &c.Scrobbles); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

func (s *Store) GetArtistAlbumStats(user string, start, end time.Time) ([]struct{Artist string; AlbumCount float64; ListenCount int64}, error) {
	query := `
		SELECT t.artist, COUNT(DISTINCT t.album) as album_count, COUNT(*) as listen_count
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.artist
		ORDER BY listen_count DESC
	`
	rows, err := s.db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []struct{Artist string; AlbumCount float64; ListenCount int64}
	for rows.Next() {
		var s struct{Artist string; AlbumCount float64; ListenCount int64}
		if err := rows.Scan(&s.Artist, &s.AlbumCount, &s.ListenCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (s *Store) GetNewArtistsCount(user string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) FROM (
			SELECT t.artist, MIN(l.date) as first_listen
			FROM Listen l JOIN Track t ON l.track = t.id
			WHERE l.user = ?
			GROUP BY t.artist
			HAVING first_listen >= ?
		)
	`
	var count int
	err := s.db.QueryRow(query, user, since.Unix()).Scan(&count)
	return count, err
}

func (s *Store) GetTotalScrobblesInPeriod(user string, start, end time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ? AND date BETWEEN ? AND ?", user, start.Unix(), end.Unix()).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
