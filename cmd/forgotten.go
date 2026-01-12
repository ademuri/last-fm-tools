package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ademuri/last-fm-tools/internal/analysis"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dormancyDays       int
	minArtistScrobbles int
	minAlbumScrobbles  int
	resultsPerBand     int
	sortBy             string
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

	forgottenCmd.Flags().IntVar(&dormancyDays, "dormancy", 90, "Minimum days since last play to qualify")
	forgottenCmd.Flags().IntVar(&minArtistScrobbles, "min-artist", 10, "Minimum scrobbles for artist inclusion")
	forgottenCmd.Flags().IntVar(&minAlbumScrobbles, "min-album", 5, "Minimum scrobbles for album inclusion")
	forgottenCmd.Flags().IntVar(&resultsPerBand, "results", 10, "Max results shown per interest band")
	forgottenCmd.Flags().StringVar(&sortBy, "sort", "dormancy", "Sort order: 'dormancy' or 'listens'")
}

func printForgotten(dbPath string) error {
	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("openDb: %w", err)
	}
	defer db.Close()

	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("checking db existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("database doesn't exist - run update first")
	}

	config := analysis.ForgottenConfig{
		DormancyDays:       dormancyDays,
		MinArtistScrobbles: minArtistScrobbles,
		MinAlbumScrobbles:  minAlbumScrobbles,
		ResultsPerBand:     resultsPerBand,
		SortBy:             sortBy,
	}

	user := viper.GetString("user")

	// 1. Forgotten Artists
	artists, err := analysis.GetForgottenArtists(db, user, config)
	if err != nil {
		return err
	}

	fmt.Println("## Forgotten Artists")
	printArtistBand(artists, analysis.BandObsession)
	printArtistBand(artists, analysis.BandStrong)
	printArtistBand(artists, analysis.BandModerate)
	fmt.Println()

	// 2. Forgotten Albums
	albums, err := analysis.GetForgottenAlbums(db, user, config)
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

	fmt.Printf("\n### %s Interest (%d+ scrobbles)\n", band, getMinScrobblesForBand(band, true))
	
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Artist", "Scrobbles", "Last Listen", "Days Ago"})
	
	for _, a := range items {
		table.Append([]string{
			a.Artist,
			strconv.FormatInt(a.TotalScrobbles, 10),
			a.LastListen.Format("2006-01-02"),
			strconv.Itoa(a.DaysSinceLast),
		})
	}
	table.Render()
}

func printAlbumBand(results map[string][]analysis.ForgottenAlbum, band string) {
	items, ok := results[band]
	if !ok || len(items) == 0 {
		return
	}

	fmt.Printf("\n### %s Interest (%d+ scrobbles)\n", band, getMinScrobblesForBand(band, false))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Artist", "Album", "Scrobbles", "Last Listen", "Days Ago"})

	for _, a := range items {
		table.Append([]string{
			a.Artist,
			a.Album,
			strconv.FormatInt(a.TotalScrobbles, 10),
			a.LastListen.Format("2006-01-02"),
			strconv.Itoa(a.DaysSinceLast),
		})
	}
	table.Render()
}

func getMinScrobblesForBand(band string, isArtist bool) int {
	if isArtist {
		switch band {
		case analysis.BandObsession:
			return 100
		case analysis.BandStrong:
			return 30
		case analysis.BandModerate:
			return 10
		}
	} else {
		switch band {
		case analysis.BandObsession:
			return 50
		case analysis.BandStrong:
			return 15
		case analysis.BandModerate:
			return 5
		}
	}
	return 0
}
