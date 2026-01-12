package analysis

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
)

// GenerateReport creates a comprehensive music taste report.
func GenerateReport(db *store.Store, user string) (*Report, error) {
	// 1. Determine Periods
	latestListen, err := db.GetLatestListen(user)
	if err != nil {
		// If no listens, default to now
		latestListen = time.Now()
	}

	// Current Period: Last 18 months
	currentEnd := latestListen
	currentStart := currentEnd.AddDate(0, -18, 0)
	
	// Historical Period: Everything before current start
	firstListen, err := db.GetFirstListen(user)
	if err != nil {
		firstListen = currentStart // Fallback
	}
	historicalStart := firstListen
	historicalEnd := currentStart

	report := &Report{}

	// 2. Metadata
	totalScrobbles, err := db.GetTotalScrobbles(user)
	if err != nil {
		return nil, fmt.Errorf("getting total scrobbles: %w", err)
	}
	totalArtists, err := db.GetTotalArtists(user)
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
	currentArtistCounts, err := db.GetTopArtists(user, currentStart, currentEnd, 30)
	if err != nil {
		return nil, fmt.Errorf("current artists: %w", err)
	}

	var currentArtists []ArtistStat
	for _, a := range currentArtistCounts {
		stat := ArtistStat{
			Name:      a.Name,
			Scrobbles: a.Scrobbles,
		}
		
		albumCounts, err := db.GetTopAlbumsForArtist(user, a.Name, currentStart, currentEnd, 3)
		if err != nil {
			return nil, err
		}
		var albums []string
		for _, ac := range albumCounts {
			albums = append(albums, fmt.Sprintf("%s (%d)", ac.Tag, ac.Count)) // reusing TagCount struct where Tag=Name
		}
		stat.TopAlbums = albums

		tags, err := db.GetTopTagsForArtist(a.Name, 3)
		if err != nil {
			return nil, err
		}
		stat.PrimaryTags = tags
		
		currentArtists = append(currentArtists, stat)
	}

	currentAlbumCounts, err := db.GetTopAlbums(user, currentStart, currentEnd, 20)
	if err != nil {
		return nil, fmt.Errorf("current albums: %w", err)
	}
	
	var currentAlbums []AlbumStat
	for _, a := range currentAlbumCounts {
		stat := AlbumStat{
			Title:     a.Title,
			Artist:    a.Artist,
			Scrobbles: a.Scrobbles,
		}
		tags, err := db.GetTopTagsForAlbum(a.Artist, a.Title, 3)
		if err != nil {
			return nil, err
		}
		stat.Tags = tags
		currentAlbums = append(currentAlbums, stat)
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
	historicalArtistCounts, err := db.GetTopArtists(user, historicalStart, historicalEnd, 30)
	if err != nil {
		return nil, fmt.Errorf("historical artists: %w", err)
	}
	
	var historicalArtists []ArtistStat
	for _, a := range historicalArtistCounts {
		stat := ArtistStat{
			Name:      a.Name,
			Scrobbles: a.Scrobbles,
		}
		
		years, err := db.GetPeakYears(user, a.Name)
		if err != nil {
			return nil, err
		}
		stat.PeakYears = years

		count, err := db.GetArtistListenCount(user, a.Name, currentStart, currentEnd)
		if err != nil {
			return nil, err
		}
		stat.InCurrentTaste = count > 0

		tags, err := db.GetTopTagsForArtist(a.Name, 3)
		if err != nil {
			return nil, err
		}
		stat.PrimaryTags = tags
		
		historicalArtists = append(historicalArtists, stat)
	}

	// Annotate Current Artists with "In Historical"
	for i := range report.CurrentTaste.TopArtists {
		count, err := db.GetArtistListenCount(user, report.CurrentTaste.TopArtists[i].Name, historicalStart, historicalEnd)
		if err != nil {
			return nil, err
		}
		report.CurrentTaste.TopArtists[i].InHistoricalBaseline = count > 0
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

	avgTracksPerAlbum, err := db.GetAverageTracksPerAlbum(user, currentStart, currentEnd)
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

var yearRegex = regexp.MustCompile(`^\d{4}$`)

func filterTags(tags []string, counts []int) []string {
	validTags := []string{}
	for i, t := range tags {
		if counts[i] < 25 {
			continue
		}
		
		normalized := strings.ToLower(t)
		normalized = strings.ReplaceAll(normalized, "-", " ")
		normalized = strings.ReplaceAll(normalized, "_", " ")
		normalized = strings.TrimSpace(normalized)

		if yearRegex.MatchString(normalized) {
			continue
		}
		
		if len(normalized) < 3 {
			continue
		}

		validTags = append(validTags, normalized)
	}
	return validTags
}

func getTopTagsWeighted(db *store.Store, user string, start, end time.Time, limit int) ([]TagStat, error) {
	// 1. Fetch all Artist Tags
	artistTagData, err := db.GetAllArtistTags()
	if err != nil {
		return nil, err
	}

	artistTagsMap := make(map[string][]string)
	
	// Group by artist
	var currentArtist string
	var currentTags []string
	var currentCounts []int
	
	for _, d := range artistTagData {
		if d.Artist != currentArtist {
			if currentArtist != "" {
				valid := filterTags(currentTags, currentCounts)
				if len(valid) >= 2 {
					artistTagsMap[currentArtist] = valid
				}
			}
			currentArtist = d.Artist
			currentTags = []string{}
			currentCounts = []int{}
		}
		currentTags = append(currentTags, d.Tag)
		currentCounts = append(currentCounts, d.Count)
	}
	if currentArtist != "" {
		valid := filterTags(currentTags, currentCounts)
		if len(valid) >= 2 {
			artistTagsMap[currentArtist] = valid
		}
	}

	// 2. Fetch all Album Tags
	albumTagData, err := db.GetAllAlbumTags()
	if err != nil {
		return nil, err
	}
	
	type albumKey struct {
		artist, album string
	}
	albumTagsMap := make(map[albumKey][]string)
	
	var currentAlbumKey albumKey
	currentTags = []string{}
	currentCounts = []int{}
	
	for _, d := range albumTagData {
		key := albumKey{d.Artist, d.Album}
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
		currentTags = append(currentTags, d.Tag)
		currentCounts = append(currentCounts, d.Count)
	}
	if currentAlbumKey.artist != "" {
		valid := filterTags(currentTags, currentCounts)
		if len(valid) >= 2 {
			albumTagsMap[currentAlbumKey] = valid
		}
	}

	// 3. Iterate Listens and Accumulate Weights
	listenCounts, err := db.GetAlbumListenCounts(user, start, end)
	if err != nil {
		return nil, err
	}

	globalTagCounts := make(map[string]int64)
	var totalWeight int64

	for _, l := range listenCounts {
		count := l.Scrobbles
		
		uniqueTags := make(map[string]bool)
		
		// Add artist tags
		if tags, ok := artistTagsMap[l.Artist]; ok {
			for _, t := range tags {
				uniqueTags[t] = true
			}
		}
		
		// Add album tags
		if tags, ok := albumTagsMap[albumKey{l.Artist, l.Title}]; ok {
			for _, t := range tags {
				uniqueTags[t] = true
			}
		}
		
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

	totalScrobblesPeriod, err := db.GetTotalScrobblesInPeriod(user, start, end)
	if err != nil {
		totalScrobblesPeriod = 1 // Fallback
	}
	if totalScrobblesPeriod == 0 {
		totalScrobblesPeriod = 1
	}

	for i := range stats {
		stats[i].Weight = stats[i].Weight / float64(totalScrobblesPeriod)
		stats[i].Weight = math.Round(stats[i].Weight*100) / 100
	}

	return stats, nil
}

func calculateDrift(histTags, currTags []TagStat) ([]DriftTag, []DriftTag) {
	histMap := make(map[string]float64)
	for _, t := range histTags {
		histMap[t.Tag] = t.Weight
	}
	currMap := make(map[string]float64)
	for _, t := range currTags {
		currMap[t.Tag] = t.Weight
	}

	histTop20 := histTags
	if len(histTop20) > 20 {
		histTop20 = histTop20[:20]
	}

	currTop20 := currTags
	if len(currTop20) > 20 {
		currTop20 = currTop20[:20]
	}
	
	var declined []DriftTag
	for _, h := range histTop20 {
		if _, exists := currMap[h.Tag]; !exists {
			declined = append(declined, DriftTag{
				Tag:              h.Tag,
				HistoricalWeight: h.Weight,
				CurrentWeight:    0,
			})
		}
	}

	var emerged []DriftTag
	for _, c := range currTop20 {
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

func calculateListeningPatterns(db *store.Store, user string, start, end time.Time) (ListeningPatterns, error) {
	lp := ListeningPatterns{}
	
	stats, err := db.GetArtistAlbumStats(user, start, end)
	if err != nil {
		return lp, err
	}

	var allCounts []float64
	var top100Counts []float64
	var artistsWith3Plus int
	
	for i, s := range stats {
		allCounts = append(allCounts, s.AlbumCount)
		
		if i < 100 {
			top100Counts = append(top100Counts, s.AlbumCount)
		}
		
		if s.AlbumCount >= 3 {
			artistsWith3Plus++
		}
	}
	
	lp.ArtistsWith3PlusAlbums = artistsWith3Plus

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

	count, err := db.GetNewArtistsCount(user, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		return lp, err
	}
	lp.NewArtistsInLast12Month = count

	totalS, _ := db.GetTotalScrobbles(user)
	totalA, _ := db.GetTotalArtists(user)
	if totalS > 0 {
		ratio := float64(totalS - int64(totalA)) / float64(totalS)
		lp.RepeatListeningRatio = math.Round(ratio*100) / 100
	}

	return lp, nil
}
