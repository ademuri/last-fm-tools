package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ademuri/last-fm-tools/internal/analysis"
	"github.com/ademuri/last-fm-tools/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	minArtistScrobbles   int
	minAlbumScrobbles    int
	resultsPerBand       int
	sortBy               string
	lastListenAfterStr   string
	lastListenBeforeStr  string
	firstListenAfterStr  string
	firstListenBeforeStr string
)

var forgottenCmd = &cobra.Command{
	Use:   "forgotten",
	Short: "Surfaces artists and albums heavily listened to in the past but not recently",
	Long:  `Identifies music that has fallen out of rotation based on dormancy and historical listen counts.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := printForgotten(viper.GetString("database"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(forgottenCmd)

	forgottenCmd.Flags().IntVar(&minArtistScrobbles, "min-artist", 10, "Minimum scrobbles for artist inclusion")
	forgottenCmd.Flags().IntVar(&minAlbumScrobbles, "min-album", 5, "Minimum scrobbles for album inclusion")
	forgottenCmd.Flags().IntVar(&resultsPerBand, "results", 10, "Max results shown per interest band")
	forgottenCmd.Flags().StringVar(&sortBy, "sort", "dormancy", "Sort order: 'dormancy' or 'listens'")
	forgottenCmd.Flags().StringVar(&lastListenAfterStr, "last_listen_after", "", "Only include entities with last listen after this date (YYYY-MM-DD)")
	forgottenCmd.Flags().StringVar(&lastListenBeforeStr, "last_listen_before", "90d", "Only include entities with last listen before this date (YYYY-MM-DD or duration like 90d)")
	forgottenCmd.Flags().StringVar(&firstListenAfterStr, "first_listen_after", "", "Only include entities with first listen after this date (YYYY-MM-DD)")
	forgottenCmd.Flags().StringVar(&firstListenBeforeStr, "first_listen_before", "", "Only include entities with first listen before this date (YYYY-MM-DD)")
}

type ForgottenAnalyzer struct {
	Config analysis.ForgottenConfig
}

func (f *ForgottenAnalyzer) Configure(params map[string]string) error {
	// Defaults
	f.Config.MinArtistScrobbles = 10
	f.Config.MinAlbumScrobbles = 5
	f.Config.ResultsPerBand = 10
	f.Config.SortBy = "dormancy"
	f.Config.LastListenBefore = time.Now().AddDate(0, 0, -90)
	f.Config.LastListenAfter = time.Unix(0, 0)
	f.Config.FirstListenBefore = time.Now()
	f.Config.FirstListenAfter = time.Unix(0, 0)

	if val, ok := params["min-artist"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid min-artist: %w", err)
		}
		f.Config.MinArtistScrobbles = v
	}
	if val, ok := params["min-album"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid min-album: %w", err)
		}
		f.Config.MinAlbumScrobbles = v
	}
	if val, ok := params["results"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid results: %w", err)
		}
		f.Config.ResultsPerBand = v
	}
	if val, ok := params["sort"]; ok {
		f.Config.SortBy = val
	}
	if val, ok := params["last_listen_before"]; ok {
		pd, err := parseSingleDatestring(val)
		if err != nil {
			return fmt.Errorf("invalid last_listen_before: %w", err)
		}
		f.Config.LastListenBefore = pd.Date
	}
	if val, ok := params["last_listen_after"]; ok {
		pd, err := parseSingleDatestring(val)
		if err != nil {
			return fmt.Errorf("invalid last_listen_after: %w", err)
		}
		f.Config.LastListenAfter = pd.Date
	}
	if val, ok := params["first_listen_before"]; ok {
		pd, err := parseSingleDatestring(val)
		if err != nil {
			return fmt.Errorf("invalid first_listen_before: %w", err)
		}
		f.Config.FirstListenBefore = pd.Date
	}
	if val, ok := params["first_listen_after"]; ok {
		pd, err := parseSingleDatestring(val)
		if err != nil {
			return fmt.Errorf("invalid first_listen_after: %w", err)
		}
		f.Config.FirstListenAfter = pd.Date
	}
	return nil
}

func (f *ForgottenAnalyzer) GetName() string {
	return "Forgotten"
}

func (f *ForgottenAnalyzer) GetResults(dbPath string, user string, start time.Time, end time.Time) (Analysis, error) {
	var a Analysis
	db, err := store.New(dbPath)
	if err != nil {
		return a, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Use config from struct, but ensure defaults if not set?
	// Configure sets defaults if called. If not called, we might have zero values.
	// But CLI usage calls SetConfig manually or we construct it.
	// We'll assume Config is set or zero values are acceptable (they are mostly 0 except dates).
	// But LastListenBefore default is 90d.
	if f.Config.LastListenBefore.IsZero() {
		f.Config.LastListenBefore = time.Now().AddDate(0, 0, -90)
	}

	artists, err := analysis.GetForgottenArtists(db, user, f.Config, time.Now())
	if err != nil {
		return a, err
	}

	albums, err := analysis.GetForgottenAlbums(db, user, f.Config, time.Now())
	if err != nil {
		return a, err
	}

	var sb strings.Builder
	sb.WriteString("<h3>Forgotten Artists</h3>")
	sb.WriteString(formatArtistBandHTML(artists, analysis.BandObsession))
	sb.WriteString(formatArtistBandHTML(artists, analysis.BandStrong))
	sb.WriteString(formatArtistBandHTML(artists, analysis.BandModerate))

	sb.WriteString("<h3>Forgotten Albums</h3>")
	sb.WriteString(formatAlbumBandHTML(albums, analysis.BandObsession))
	sb.WriteString(formatAlbumBandHTML(albums, analysis.BandStrong))
	sb.WriteString(formatAlbumBandHTML(albums, analysis.BandModerate))

	a.BodyOverride = sb.String()
	return a, nil
}

func formatArtistBandHTML(results map[string][]analysis.ForgottenArtist, band string) string {
	items, ok := results[band]
	if !ok || len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h4>%s Interest (%d+ scrobbles)</h4>", band, analysis.GetThreshold(band, true)))
	sb.WriteString("<table><thead><tr><th>Artist</th><th>Scrobbles</th><th>Last Listen</th></tr></thead><tbody>")
	for _, a := range items {
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%s</td></tr>",
			a.Artist, a.TotalScrobbles, a.LastListen.Format("2006-01-02")))
	}
	sb.WriteString("</tbody></table>")
	return sb.String()
}

func formatAlbumBandHTML(results map[string][]analysis.ForgottenAlbum, band string) string {
	items, ok := results[band]
	if !ok || len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h4>%s Interest (%d+ scrobbles)</h4>", band, analysis.GetThreshold(band, false)))
	sb.WriteString("<table><thead><tr><th>Artist</th><th>Album</th><th>Scrobbles</th><th>Last Listen</th></tr></thead><tbody>")
	for _, a := range items {
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>",
			a.Artist, a.Album, a.TotalScrobbles, a.LastListen.Format("2006-01-02")))
	}
	sb.WriteString("</tbody></table>")
	return sb.String()
}

func printForgotten(dbPath string) error {
	// Determine time range from global flags
	var lastListenBefore time.Time
	if lastListenBeforeStr != "" {
		pd, err := parseSingleDatestring(lastListenBeforeStr)
		if err != nil {
			return fmt.Errorf("invalid last_listen_before date: %w", err)
		}
		lastListenBefore = pd.Date
	} else {
		lastListenBefore = time.Now().AddDate(0, 0, -90)
	}

	var lastListenAfter time.Time
	if lastListenAfterStr != "" {
		pd, err := parseSingleDatestring(lastListenAfterStr)
		if err != nil {
			return fmt.Errorf("invalid last_listen_after date: %w", err)
		}
		lastListenAfter = pd.Date
	} else {
		lastListenAfter = time.Unix(0, 0)
	}

	var firstListenBefore time.Time
	if firstListenBeforeStr != "" {
		pd, err := parseSingleDatestring(firstListenBeforeStr)
		if err != nil {
			return fmt.Errorf("invalid first_listen_before date: %w", err)
		}
		firstListenBefore = pd.Date
	} else {
		firstListenBefore = time.Now()
	}

	var firstListenAfter time.Time
	if firstListenAfterStr != "" {
		pd, err := parseSingleDatestring(firstListenAfterStr)
		if err != nil {
			return fmt.Errorf("invalid first_listen_after date: %w", err)
		}
		firstListenAfter = pd.Date
	} else {
		firstListenAfter = time.Unix(0, 0)
	}

	config := analysis.ForgottenConfig{
		LastListenAfter:    lastListenAfter,
		LastListenBefore:   lastListenBefore,
		FirstListenAfter:   firstListenAfter,
		FirstListenBefore:  firstListenBefore,
		MinArtistScrobbles: minArtistScrobbles,
		MinAlbumScrobbles:  minAlbumScrobbles,
		ResultsPerBand:     resultsPerBand,
		SortBy:             sortBy,
	}

	db, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	user := viper.GetString("user")

	// 1. Forgotten Artists
	artists, err := analysis.GetForgottenArtists(db, user, config, time.Now())
	if err != nil {
		return err
	}

	fmt.Println("## Forgotten Artists")
	printArtistBand(artists, analysis.BandObsession)
	printArtistBand(artists, analysis.BandStrong)
	printArtistBand(artists, analysis.BandModerate)
	fmt.Println()

	// 2. Forgotten Albums
	albums, err := analysis.GetForgottenAlbums(db, user, config, time.Now())
	if err != nil {
		return err
	}

	fmt.Println("## Forgotten Albums")
	printAlbumBand(albums, analysis.BandObsession)
	printAlbumBand(albums, analysis.BandStrong)
	printAlbumBand(albums, analysis.BandModerate)

	return nil
}

func printArtistBand(results map[string][]analysis.ForgottenArtist, band string) {
	items, ok := results[band]
	if !ok || len(items) == 0 {
		return
	}

	fmt.Printf("\n### %s Interest (%d+ scrobbles)\n", band, analysis.GetThreshold(band, true))
	
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Artist", "Scrobbles", "Last Listen"})
	
	for _, a := range items {
		table.Append([]string{
			a.Artist,
			strconv.FormatInt(a.TotalScrobbles, 10),
			a.LastListen.Format("2006-01-02"),
		})
	}
	table.Render()
}

func printAlbumBand(results map[string][]analysis.ForgottenAlbum, band string) {
	items, ok := results[band]
	if !ok || len(items) == 0 {
		return
	}

	fmt.Printf("\n### %s Interest (%d+ scrobbles)\n", band, analysis.GetThreshold(band, false))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Artist", "Album", "Scrobbles", "Last Listen"})

	for _, a := range items {
		table.Append([]string{
			a.Artist,
			a.Album,
			strconv.FormatInt(a.TotalScrobbles, 10),
			a.LastListen.Format("2006-01-02"),
		})
	}
	table.Render()
}