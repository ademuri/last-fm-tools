package analysis

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

// GenerateReport creates a comprehensive music taste report.
func GenerateReport(db *sql.DB, user string) (*Report, error) {
	// 1. Determine Periods
	latestListen, err := getLatestListen(db, user)
	if err != nil {
		// If no listens, default to now
		latestListen = time.Now()
	}

	// Current Period: Last 18 months
	currentEnd := latestListen
	currentStart := currentEnd.AddDate(0, -18, 0)
	
	// Historical Period: Everything before current start
	// We need the very first listen to define the start of history
	firstListen, err := getFirstListen(db, user)
	if err != nil {
		firstListen = currentStart // Fallback
	}
	historicalStart := firstListen
	historicalEnd := currentStart

	report := &Report{}

	// 2. Metadata
	totalScrobbles, err := getTotalScrobbles(db, user)
	if err != nil {
		return nil, fmt.Errorf("getting total scrobbles: %w", err)
	}
	totalArtists, err := getTotalArtists(db, user)
	if err != nil {
		return nil, fmt.Errorf("getting total artists: %w", err)
	}

	report.Metadata = ProfileMetadata{
		GeneratedDate:    time.Now().Format("2006-01-02"),
		TotalScrobbles:   totalScrobbles,
		TotalArtists:     totalArtists,
		CurrentPeriod:    fmt.Sprintf("%s to %s", currentStart.Format("2006-01-02"), currentEnd.Format("2006-01-02")),
		HistoricalPeriod: fmt.Sprintf("%s to %s", historicalStart.Format("2006-01-02"), historicalEnd.Format("2006-01-02")),
	}

	// 3. Current Taste
	currentArtists, err := getTopArtists(db, user, currentStart, currentEnd, 30)
	if err != nil {
		return nil, fmt.Errorf("current artists: %w", err)
	}
	// Enrich current artists with top albums/tags
	for i := range currentArtists {
		albums, err := getTopAlbumsForArtist(db, user, currentArtists[i].Name, currentStart, currentEnd, 3)
		if err != nil {
			return nil, err
		}
		currentArtists[i].TopAlbums = albums
		
		tags, err := getTopTagsForArtist(db, currentArtists[i].Name, 3)
		if err != nil {
			return nil, err
		}
		currentArtists[i].PrimaryTags = tags
	}

	currentAlbums, err := getTopAlbums(db, user, currentStart, currentEnd, 20)
	if err != nil {
		return nil, fmt.Errorf("current albums: %w", err)
	}
	// Enrich albums with tags
	for i := range currentAlbums {
		tags, err := getTopTagsForAlbum(db, currentAlbums[i].Artist, currentAlbums[i].Title, 3)
		if err != nil {
			return nil, err
		}
		currentAlbums[i].Tags = tags
	}

	currentTags, err := getTopTagsWeighted(db, user, currentStart, currentEnd, 40)
	if err != nil {
		return nil, fmt.Errorf("current tags: %w", err)
	}

	report.CurrentTaste = TasteProfile{
		TopArtists: currentArtists,
		TopAlbums:  currentAlbums,
		TopTags:    currentTags,
	}

	// 4. Historical Baseline
	historicalArtists, err := getTopArtists(db, user, historicalStart, historicalEnd, 30)
	if err != nil {
		return nil, fmt.Errorf("historical artists: %w", err)
	}
	for i := range historicalArtists {
		years, err := getPeakYears(db, user, historicalArtists[i].Name)
		if err != nil {
			return nil, err
		}
		historicalArtists[i].PeakYears = years

		count, err := getArtistListenCount(db, user, historicalArtists[i].Name, currentStart, currentEnd)
		if err != nil {
			return nil, err
		}
		historicalArtists[i].InCurrentTaste = count > 0

		tags, err := getTopTagsForArtist(db, historicalArtists[i].Name, 3)
		if err != nil {
			return nil, err
		}
		historicalArtists[i].PrimaryTags = tags
	}

	// Annotate Current Artists with "In Historical"
	for i := range report.CurrentTaste.TopArtists {
		inHist := false
		// We only fetched top 30 historical. 
		// Ideally we should check if they existed in history at all or were in the top X.
		// Prompt says "in_historical_baseline". Let's assume this refers to the *Top 30* baseline we generated?
		// Or just if they had listens in the historical period.
		// Let's check if they had significant listens in history.
		// Creating a helper for "count in period" is better.
		count, err := getArtistListenCount(db, user, report.CurrentTaste.TopArtists[i].Name, historicalStart, historicalEnd)
		if err != nil {
			return nil, err
		}
		inHist = count > 0 // Or some threshold
		report.CurrentTaste.TopArtists[i].InHistoricalBaseline = inHist
	}

	historicalTags, err := getTopTagsWeighted(db, user, historicalStart, historicalEnd, 40)
	if err != nil {
		return nil, fmt.Errorf("historical tags: %w", err)
	}

	report.HistoricalBaseline = TasteProfile{
		TopArtists: historicalArtists,
		TopTags:    historicalTags,
	}

	// 5. Taste Drift
	declined, emerged := calculateDrift(historicalTags, currentTags)
	report.TasteDrift = TasteDrift{
		DeclinedTags: declined,
		EmergedTags:  emerged,
	}

	// 6. Listening Patterns
	lp, err := calculateListeningPatterns(db, user, currentStart, currentEnd)
	if err != nil {
		return nil, fmt.Errorf("listening patterns: %w", err)
	}
	report.ListeningPatterns = lp

	// Calculate Listening Style (Profile Metadata) based on patterns
	// Heuristic: "Tracks per Album" is a better indicator of style than "Albums per Artist".
	// If avg tracks per album > 3.0, likely album-oriented.
	avgTracksPerAlbum, err := getAverageTracksPerAlbum(db, user, currentStart, currentEnd)
	if err != nil {
		return nil, fmt.Errorf("getting avg tracks per album: %w", err)
	}

	if avgTracksPerAlbum >= 3.0 {
		report.Metadata.ListeningStyle = "album-oriented"
	} else {
		report.Metadata.ListeningStyle = "track-oriented"
	}

	return report, nil
}

// -- Helpers --

func getAverageTracksPerAlbum(db *sql.DB, user string, start, end time.Time) (float64, error) {
	query := `
		SELECT COUNT(DISTINCT t.id)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.artist, t.album
	`
	rows, err := db.Query(query, user, start.Unix(), end.Unix())
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

func getLatestListen(db *sql.DB, user string) (time.Time, error) {
	var date int64
	err := db.QueryRow("SELECT MAX(date) FROM Listen WHERE user = ?", user).Scan(&date)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(date, 0), nil
}

func getFirstListen(db *sql.DB, user string) (time.Time, error) {
	var date int64
	err := db.QueryRow("SELECT MIN(date) FROM Listen WHERE user = ?", user).Scan(&date)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(date, 0), nil
}

func getTotalScrobbles(db *sql.DB, user string) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ?", user).Scan(&count)
	return count, err
}

func getTotalArtists(db *sql.DB, user string) (int, error) {
	var count int
	// We count artists from Listen table to ensure they are actually listened to by user
	query := `SELECT COUNT(DISTINCT t.artist) FROM Listen l JOIN Track t ON l.track = t.id WHERE l.user = ?`
	err := db.QueryRow(query, user).Scan(&count)
	return count, err
}

func getTopArtists(db *sql.DB, user string, start, end time.Time, limit int) ([]ArtistStat, error) {
	query := `
		SELECT t.artist, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ?
		GROUP BY t.artist
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := db.Query(query, user, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []ArtistStat
	for rows.Next() {
		var a ArtistStat
		if err := rows.Scan(&a.Name, &a.Scrobbles); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, nil
}

func getTopAlbumsForArtist(db *sql.DB, user, artist string, start, end time.Time, limit int) ([]string, error) {
	query := `
		SELECT t.album, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.album
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := db.Query(query, user, artist, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []string
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		// Prompt format: "Album Name (year)" - Note: we don't have year in DB usually, just name.
		// DB schema has Album table, but no year column in create-tables.sql.
		// So we just return Name.
		albums = append(albums, name)
	}
	return albums, nil
}

func getTopTagsForArtist(db *sql.DB, artist string, limit int) ([]string, error) {
	// Use ArtistTag table, order by count (Last.fm popularity)
	query := `SELECT tag FROM ArtistTag WHERE artist = ? ORDER BY count DESC LIMIT ?`
	rows, err := db.Query(query, artist, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		rows.Scan(&tag)
		tags = append(tags, tag)
	}
	return tags, nil
}

func getTopAlbums(db *sql.DB, user string, start, end time.Time, limit int) ([]AlbumStat, error) {
	query := `
		SELECT t.album, t.artist, COUNT(*) as scrobbles
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.album, t.artist
		ORDER BY scrobbles DESC
		LIMIT ?
	`
	rows, err := db.Query(query, user, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []AlbumStat
	for rows.Next() {
		var a AlbumStat
		if err := rows.Scan(&a.Title, &a.Artist, &a.Scrobbles); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func getTopTagsForAlbum(db *sql.DB, artist, album string, limit int) ([]string, error) {
	query := `SELECT tag FROM AlbumTag WHERE artist = ? AND album = ? ORDER BY count DESC LIMIT ?`
	rows, err := db.Query(query, artist, album, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tags []string
	for rows.Next() {
		var tag string
		rows.Scan(&tag)
		tags = append(tags, tag)
	}
	return tags, nil
}

var yearRegex = regexp.MustCompile(`^\d{4}$`)

func filterTags(tags []string, counts []int) []string {
	validTags := []string{}
	for i, t := range tags {
		// 1. Weight >= 25
		if counts[i] < 25 {
			continue
		}
		
		// 4. Normalize
		normalized := strings.ToLower(t)
		normalized = strings.ReplaceAll(normalized, "-", " ")
		normalized = strings.ReplaceAll(normalized, "_", " ")
		normalized = strings.TrimSpace(normalized)

		// 2. Discard year pattern
		if yearRegex.MatchString(normalized) {
			continue
		}
		
		// 3. Length < 3
		if len(normalized) < 3 {
			continue
		}

		validTags = append(validTags, normalized)
	}
	return validTags
}

func getTopTagsWeighted(db *sql.DB, user string, start, end time.Time, limit int) ([]TagStat, error) {
	// 1. Fetch all Artist Tags
	artistTagsMap := make(map[string][]string)
	rows, err := db.Query("SELECT artist, tag, count FROM ArtistTag")
	if err != nil {
		return nil, err
	}
	
	var currentArtist string
	var currentTags []string
	var currentCounts []int
	
	for rows.Next() {
		var a, t string
		var c int
		if err := rows.Scan(&a, &t, &c); err != nil {
			rows.Close()
			return nil, err
		}
		
		if a != currentArtist {
			if currentArtist != "" {
				valid := filterTags(currentTags, currentCounts)
				if len(valid) >= 2 {
					artistTagsMap[currentArtist] = valid
				}
			}
			currentArtist = a
			currentTags = []string{}
			currentCounts = []int{}
		}
		currentTags = append(currentTags, t)
		currentCounts = append(currentCounts, c)
	}
	// Process last
	if currentArtist != "" {
		valid := filterTags(currentTags, currentCounts)
		if len(valid) >= 2 {
			artistTagsMap[currentArtist] = valid
		}
	}
	rows.Close()

	// 2. Fetch all Album Tags
	type albumKey struct {
		artist, album string
	}
	albumTagsMap := make(map[albumKey][]string)
	
	rows, err = db.Query("SELECT artist, album, tag, count FROM AlbumTag ORDER BY artist, album")
	if err != nil {
		return nil, err
	}
	
	var currentAlbumKey albumKey
	currentTags = []string{}
	currentCounts = []int{}
	
	for rows.Next() {
		var a, alb, t string
		var c int
		if err := rows.Scan(&a, &alb, &t, &c); err != nil {
			rows.Close()
			return nil, err
		}
		
		key := albumKey{a, alb}
		if key != currentAlbumKey {
			if currentAlbumKey.artist != "" {
				valid := filterTags(currentTags, currentCounts)
				if len(valid) >= 2 {
					albumTagsMap[currentAlbumKey] = valid
				}
			}
			currentAlbumKey = key
			currentTags = []string{}
			currentCounts = []int{}
		}
		currentTags = append(currentTags, t)
		currentCounts = append(currentCounts, c)
	}
	// Process last
	if currentAlbumKey.artist != "" {
		valid := filterTags(currentTags, currentCounts)
		if len(valid) >= 2 {
			albumTagsMap[currentAlbumKey] = valid
		}
	}
	rows.Close()

	// 3. Iterate Listens and Accumulate Weights
	query := `
		SELECT t.artist, t.album, COUNT(*)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ?
		GROUP BY t.artist, t.album
	`
	rows, err = db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	globalTagCounts := make(map[string]int64)
	var totalWeight int64

	for rows.Next() {
		var artist, album string
		var count int64
		if err := rows.Scan(&artist, &album, &count); err != nil {
			return nil, err
		}

		// Gather unique tags for this listen context
		uniqueTags := make(map[string]bool)
		
		// Add artist tags
		if tags, ok := artistTagsMap[artist]; ok {
			for _, t := range tags {
				uniqueTags[t] = true
			}
		}
		
		// Add album tags
		if tags, ok := albumTagsMap[albumKey{artist, album}]; ok {
			for _, t := range tags {
				uniqueTags[t] = true
			}
		}
		
		// Add to global
		for t := range uniqueTags {
			globalTagCounts[t] += count
			totalWeight += count
		}
	}

	// 4. Convert to TagStat and Sort
	var stats []TagStat
	for tag, weight := range globalTagCounts {
		stats = append(stats, TagStat{Tag: tag, Weight: float64(weight)})
	}
	
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Weight > stats[j].Weight
	})
	
	if len(stats) > limit {
		stats = stats[:limit]
	}

	// Normalize (relative frequency 0-1 based on total tag-assignments)
	// Or relative to the max? Prompt example says "normalized 0-1, relative frequency".
	// Example: tag: "electronic", weight: 0.85. 
	// If it's relative frequency, sum should be 1.0 (or close).
	// But 0.85 + 0.72 > 1.0. So it's likely normalized against the *Top Tag* or just a score.
	// Prompt says: "normalized 0-1, relative frequency". 
	// Let's interpret as: weight / total_scrobbles? Or weight / max_weight?
	// If "electronic" is 0.85, it means 85% of listens had this tag? That makes sense.
	// So we normalize by Total Scrobbles in the period? No, total scrobbles *that had tags*?
	// Let's normalize by Total Scrobbles in period (approximate).
	// Actually, let's normalize by the count of the top tag? No, 0.85 implies percentage.
	// Let's divide by Total Scrobbles in the period.
	
	totalScrobblesPeriod := int64(0)
	err = db.QueryRow("SELECT COUNT(*) FROM Listen WHERE user = ? AND date BETWEEN ? AND ?", user, start.Unix(), end.Unix()).Scan(&totalScrobblesPeriod)
	if err != nil {
		// Fallback
		totalScrobblesPeriod = 1
	}

	for i := range stats {
		if totalScrobblesPeriod > 0 {
			stats[i].Weight = stats[i].Weight / float64(totalScrobblesPeriod)
			stats[i].Weight = math.Round(stats[i].Weight*100) / 100
		}
	}

	return stats, nil
}

func getArtistListenCount(db *sql.DB, user, artist string, start, end time.Time) (int64, error) {
	query := `
		SELECT COUNT(*) 
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ? AND l.date BETWEEN ? AND ?
	`
	var count int64
	err := db.QueryRow(query, user, artist, start.Unix(), end.Unix()).Scan(&count)
	return count, err
}

func getPeakYears(db *sql.DB, user, artist string) (string, error) {
	query := `
		SELECT strftime('%Y', datetime(date, 'unixepoch')) as year, COUNT(*)
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND t.artist = ?
		GROUP BY year
		ORDER BY year
	`
	rows, err := db.Query(query, user, artist)
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

	target := int(float64(total) * 0.8)
	
	// Find shortest continuous range >= target
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

func calculateDrift(histTags, currTags []TagStat) ([]DriftTag, []DriftTag) {
	// Declined: Hist Top 20 AND NOT Curr Top 40
	// Emerged: Curr Top 20 AND NOT Hist Top 40
	
	histMap := make(map[string]float64)
	for _, t := range histTags {
		histMap[t.Tag] = t.Weight
	}
	currMap := make(map[string]float64)
	for _, t := range currTags {
		currMap[t.Tag] = t.Weight
	}

	// Hist top 20
	histTop20 := histTags
	if len(histTop20) > 20 {
		histTop20 = histTop20[:20]
	}

	// Curr top 20
	currTop20 := currTags
	if len(currTop20) > 20 {
		currTop20 = currTop20[:20]
	}
	
	// Check thresholds for existence (Top 40 is what we fetched)
	
	var declined []DriftTag
	for _, h := range histTop20 {
		// If not in current (Top 40 fetched)
		if _, exists := currMap[h.Tag]; !exists {
			declined = append(declined, DriftTag{
				Tag:              h.Tag,
				HistoricalWeight: h.Weight,
				CurrentWeight:    0, // effectively 0 in top 40
			})
		}
	}

	var emerged []DriftTag
	for _, c := range currTop20 {
		// If not in historical (Top 40 fetched)
		if _, exists := histMap[c.Tag]; !exists {
			emerged = append(emerged, DriftTag{
				Tag:              c.Tag,
				HistoricalWeight: 0,
				CurrentWeight:    c.Weight,
			})
		}
	}
	
	return declined, emerged
}

func calculateListeningPatterns(db *sql.DB, user string, start, end time.Time) (ListeningPatterns, error) {
	lp := ListeningPatterns{}

	// We need album counts per artist, but also need to know who the top artists are (by listens)
	// to filter for Top 100.
	
	// 1. Get album counts and listen counts per artist
	query := `
		SELECT t.artist, COUNT(DISTINCT t.album) as album_count, COUNT(*) as listen_count
		FROM Listen l
		JOIN Track t ON l.track = t.id
		WHERE l.user = ? AND l.date BETWEEN ? AND ? AND t.album != ''
		GROUP BY t.artist
		ORDER BY listen_count DESC
	`
	rows, err := db.Query(query, user, start.Unix(), end.Unix())
	if err != nil {
		return lp, err
	}
	defer rows.Close()

	var allCounts []float64
	var top100Counts []float64
	var artistsWith3Plus int
	
	i := 0
	for rows.Next() {
		var artist string
		var albums float64
		var listens int64
		if err := rows.Scan(&artist, &albums, &listens); err != nil {
			return lp, err
		}
		
		allCounts = append(allCounts, albums)
		
		if i < 100 {
			top100Counts = append(top100Counts, albums)
		}
		
		if albums >= 3 {
			artistsWith3Plus++
		}
		i++
	}
	
	lp.ArtistsWith3PlusAlbums = artistsWith3Plus

	// Helper to calc median/avg
	calcStats := func(counts []float64) (median float64, avg float64) {
		if len(counts) == 0 {
			return 0, 0
		}
		sum := 0.0
		for _, c := range counts {
			sum += c
		}
		avg = math.Round((sum / float64(len(counts)))*10) / 10
		
		sort.Float64s(counts)
		mid := len(counts) / 2
		if len(counts)%2 == 1 {
			median = counts[mid]
		} else {
			median = (counts[mid-1] + counts[mid]) / 2
		}
		return
	}

	lp.AllAlbumsPerArtistMedian, lp.AllAlbumsPerArtistAverage = calcStats(allCounts)
	lp.Top100ArtistsAlbumsMedian, lp.Top100ArtistsAlbumsAverage = calcStats(top100Counts)

	// New artists in past 12 months
	// Count artists whose First Listen is > (Now - 12m)
	newArtistsStart := time.Now().AddDate(-1, 0, 0)
	queryDisc := `
		SELECT COUNT(*) FROM (
			SELECT t.artist, MIN(l.date) as first_listen
			FROM Listen l JOIN Track t ON l.track = t.id
			WHERE l.user = ?
			GROUP BY t.artist
			HAVING first_listen >= ?
		)
	`
	db.QueryRow(queryDisc, user, newArtistsStart.Unix()).Scan(&lp.NewArtistsInLast12Month)

	// Repeat Ratio
	// (TotalScrobbles - UniqueArtists) / TotalScrobbles
	totalS, _ := getTotalScrobbles(db, user)
	totalA, _ := getTotalArtists(db, user)
	if totalS > 0 {
		ratio := float64(totalS - int64(totalA)) / float64(totalS)
		lp.RepeatListeningRatio = math.Round(ratio*100) / 100
	}

	return lp, nil
}