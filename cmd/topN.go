/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	limitArtists int
	limitAlbums  int
	limitTracks  int
)

var topNCmd = &cobra.Command{
	Use:   "top-n [from] [to (optional)]",
	Short: "Generates a textual summary of music taste",
	Long:  `Generates a comprehensive report including top artists and albums over a specified period.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := printTopN(viper.GetString("database"), args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(topNCmd)
	topNCmd.Flags().IntVar(&limitArtists, "artists", 10, "Number of top artists to show")
	topNCmd.Flags().IntVar(&limitAlbums, "albums", 10, "Number of top albums to show")
	topNCmd.Flags().IntVar(&limitTracks, "tracks", 10, "Number of top tracks to show")
}

func printTopN(dbPath string, args []string) error {
	start, end, err := parseDateRangeFromArgs(args)
	if err != nil {
		return err
	}
	user := viper.GetString("user")

	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("openDb: %w", err)
	}

	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("dbExists: %w", err)
	}
	if !exists {
		return fmt.Errorf("Database doesn't exist - run update first.")
	}

	// 1. Total Scrobbles
	var totalScrobbles int64
	err = db.QueryRow("SELECT COUNT(id) FROM Listen WHERE user = ? AND date BETWEEN ? AND ?", user, start.Unix(), end.Unix()).Scan(&totalScrobbles)
	if err != nil {
		return fmt.Errorf("counting total scrobbles: %w", err)
	}

	fmt.Printf("Music Taste Report for User: %s\n", user)
	fmt.Printf("Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("Total Scrobbles: %d\n\n", totalScrobbles)

	// 2. Top Artists
	if limitArtists > 0 {
		const artistQueryString = `
		SELECT Track.artist, COUNT(Listen.id)
		FROM Listen
		INNER JOIN Track ON Track.id = Listen.track
		WHERE user = ?
		AND Listen.date BETWEEN ? AND ?
		GROUP BY Track.artist
		ORDER BY COUNT(*) DESC
		LIMIT ?
		;
		`
		artistRows, err := db.Query(artistQueryString, user, start.Unix(), end.Unix(), limitArtists)
		if err != nil {
			return fmt.Errorf("querying artists: %w", err)
		}
		defer artistRows.Close()

		fmt.Printf("## Top %d Artists\n", limitArtists)
		i := 1
		for artistRows.Next() {
			var name string
			var count int64
			if err := artistRows.Scan(&name, &count); err != nil {
				return fmt.Errorf("scanning artist: %w", err)
			}
			fmt.Printf("%d. %s (%d)\n", i, name, count)
			i++
		}
		fmt.Println()
	}

	// 3. Top Albums
	if limitAlbums > 0 {
		const albumQueryString = `
		SELECT Track.artist, Track.album, COUNT(Listen.id)
		FROM Listen
		INNER JOIN Track ON Track.id = Listen.track
		WHERE user = ?
		AND Listen.date BETWEEN ? AND ?
		GROUP BY Track.artist, Track.album
		ORDER BY COUNT(*) DESC
		LIMIT ?
		;
		`
		albumRows, err := db.Query(albumQueryString, user, start.Unix(), end.Unix(), limitAlbums)
		if err != nil {
			return fmt.Errorf("querying albums: %w", err)
		}
		defer albumRows.Close()

		fmt.Printf("## Top %d Albums\n", limitAlbums)
		i := 1
		for albumRows.Next() {
			var artist, album string
			var count int64
			if err := albumRows.Scan(&artist, &album, &count); err != nil {
				return fmt.Errorf("scanning album: %w", err)
			}
			fmt.Printf("%d. %s - %s (%d)\n", i, album, artist, count)
			i++
		}
		fmt.Println()
	}

	// 4. Top Tracks
	if limitTracks > 0 {
		const trackQueryString = `
		SELECT Track.name, Track.artist, COUNT(Listen.id)
		FROM Listen
		INNER JOIN Track ON Track.id = Listen.track
		WHERE user = ?
		AND Listen.date BETWEEN ? AND ?
		GROUP BY Track.name, Track.artist
		ORDER BY COUNT(*) DESC
		LIMIT ?
		;
		`
		trackRows, err := db.Query(trackQueryString, user, start.Unix(), end.Unix(), limitTracks)
		if err != nil {
			return fmt.Errorf("querying tracks: %w", err)
		}
		defer trackRows.Close()

		fmt.Printf("## Top %d Tracks\n", limitTracks)
		i := 1
		for trackRows.Next() {
			var name, artist string
			var count int64
			if err := trackRows.Scan(&name, &artist, &count); err != nil {
				return fmt.Errorf("scanning track: %w", err)
			}
			fmt.Printf("%d. %s - %s (%d)\n", i, name, artist, count)
			i++
		}
		fmt.Println()
	}

	return nil
}
