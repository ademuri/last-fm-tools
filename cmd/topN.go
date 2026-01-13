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
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	limitArtists int
	limitAlbums  int
	limitTracks  int
	limitTags    int
)

var topNCmd = &cobra.Command{
	Use:   "top-n [from] [to (optional)]",
	Short: "Generates a textual summary of music taste",
	Long:  `Generates a comprehensive report including top artists and albums over a specified period.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		start, end, err := parseDateRangeFromArgs(args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		analyzer := &TopNAnalyzer{}
		// Set config from flags
		params := map[string]string{
			"artists": strconv.Itoa(limitArtists),
			"albums":  strconv.Itoa(limitAlbums),
			"tracks":  strconv.Itoa(limitTracks),
			"tags":    strconv.Itoa(limitTags),
		}
		analyzer.Configure(params)

		// GetResults returns Analysis with BodyOverride (HTML)
		// For CLI we want text.
		// So we might need to keep printTopN or make TopNAnalyzer support text output too?
		// Or just stick with printTopN for CLI as they are quite different.
		// I will keep printTopN for CLI usage as it writes to io.Writer and formatting is different (markdown-ish).
		err = printTopN(os.Stdout, viper.GetString("database"), start, end, limitArtists, limitAlbums, limitTracks, limitTags)
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
	topNCmd.Flags().IntVar(&limitTags, "tags", 5, "Number of top tags to show for artists and albums")
}

type TopNAnalyzer struct {
	LimitArtists int
	LimitAlbums  int
	LimitTracks  int
	LimitTags    int
}

func (t *TopNAnalyzer) Configure(params map[string]string) error {
	t.LimitArtists = 10
	t.LimitAlbums = 10
	t.LimitTracks = 10
	t.LimitTags = 5

	if val, ok := params["artists"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		t.LimitArtists = v
	}
	if val, ok := params["albums"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		t.LimitAlbums = v
	}
	if val, ok := params["tracks"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		t.LimitTracks = v
	}
	if val, ok := params["tags"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		t.LimitTags = v
	}
	return nil
}

func (t *TopNAnalyzer) GetName() string {
	return "Top N Report"
}

func (t *TopNAnalyzer) GetResults(dbPath string, user string, start time.Time, end time.Time) (Analysis, error) {
	var a Analysis
	db, err := openDb(dbPath)
	if err != nil {
		return a, fmt.Errorf("openDb: %w", err)
	}
	defer db.Close()

	var sb strings.Builder

	// 1. Top Artists
	if t.LimitArtists > 0 {
		rows, err := db.Query(`
		SELECT Track.artist, COUNT(Listen.id)
		FROM Listen
		INNER JOIN Track ON Track.id = Listen.track
		WHERE user = ? AND Listen.date BETWEEN ? AND ?
		GROUP BY Track.artist
		ORDER BY COUNT(*) DESC
		LIMIT ?`, user, start.Unix(), end.Unix(), t.LimitArtists)
		if err != nil {
			return a, err
		}
		defer rows.Close()

		sb.WriteString(fmt.Sprintf("<h3>Top %d Artists</h3>", t.LimitArtists))
		sb.WriteString("<table><thead><tr><th>Rank</th><th>Artist</th><th>Scrobbles</th><th>Tags</th></tr></thead><tbody>")
		i := 1
		for rows.Next() {
			var name string
			var count int64
			if err := rows.Scan(&name, &count); err != nil {
				return a, err
			}
			tags, _ := getArtistTags(db, name, t.LimitTags)
			sb.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%d</td><td>%s</td></tr>", i, name, count, tags))
			i++
		}
		sb.WriteString("</tbody></table>")
		rows.Close()
	}

	// 2. Top Albums
	if t.LimitAlbums > 0 {
		rows, err := db.Query(`
		SELECT Track.artist, Track.album, COUNT(Listen.id)
		FROM Listen
		INNER JOIN Track ON Track.id = Listen.track
		WHERE user = ? AND Listen.date BETWEEN ? AND ?
		GROUP BY Track.artist, Track.album
		ORDER BY COUNT(*) DESC
		LIMIT ?`, user, start.Unix(), end.Unix(), t.LimitAlbums)
		if err != nil {
			return a, err
		}
		defer rows.Close()

		sb.WriteString(fmt.Sprintf("<h3>Top %d Albums</h3>", t.LimitAlbums))
		sb.WriteString("<table><thead><tr><th>Rank</th><th>Album</th><th>Artist</th><th>Scrobbles</th><th>Tags</th></tr></thead><tbody>")
		i := 1
		for rows.Next() {
			var artist, album string
			var count int64
			if err := rows.Scan(&artist, &album, &count); err != nil {
				return a, err
			}
			tags, _ := getAlbumTags(db, artist, album, t.LimitTags)
			sb.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>", i, album, artist, count, tags))
			i++
		}
		sb.WriteString("</tbody></table>")
		rows.Close()
	}

	a.BodyOverride = sb.String()
	return a, nil
}

func printTopN(out io.Writer, dbPath string, start, end time.Time, limitArtists, limitAlbums, limitTracks, limitTags int) error {
	user := viper.GetString("user")

	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("openDb: %w", err)
	}
	defer db.Close()

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

	fmt.Fprintf(out, "Music Taste Report for User: %s\n", user)
	fmt.Fprintf(out, "Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Fprintf(out, "Total Scrobbles: %d\n\n", totalScrobbles)

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

		fmt.Fprintf(out, "## Top %d Artists\n", limitArtists)
		i := 1
		for artistRows.Next() {
			var name string
			var count int64
			if err := artistRows.Scan(&name, &count); err != nil {
				return fmt.Errorf("scanning artist: %w", err)
			}

			tags, err := getArtistTags(db, name, limitTags)
			if err != nil {
				return fmt.Errorf("getting artist tags: %w", err)
			}

			if tags != "" {
				fmt.Fprintf(out, "%d. %s (%d) - [%s]\n", i, name, count, tags)
			} else {
				fmt.Fprintf(out, "%d. %s (%d)\n", i, name, count)
			}
			i++
		}
		fmt.Fprintln(out)
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

		fmt.Fprintf(out, "## Top %d Albums\n", limitAlbums)
		i := 1
		for albumRows.Next() {
			var artist, album string
			var count int64
			if err := albumRows.Scan(&artist, &album, &count); err != nil {
				return fmt.Errorf("scanning album: %w", err)
			}

			tags, err := getAlbumTags(db, artist, album, limitTags)
			if err != nil {
				return fmt.Errorf("getting album tags: %w", err)
			}

			if tags != "" {
				fmt.Fprintf(out, "%d. %s - %s (%d) - [%s]\n", i, album, artist, count, tags)
			} else {
				fmt.Fprintf(out, "%d. %s - %s (%d)\n", i, album, artist, count)
			}
			i++
		}
		fmt.Fprintln(out)
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

		fmt.Fprintf(out, "## Top %d Tracks\n", limitTracks)
		i := 1
		for trackRows.Next() {
			var name, artist string
			var count int64
			if err := trackRows.Scan(&name, &artist, &count); err != nil {
				return fmt.Errorf("scanning track: %w", err)
			}
			fmt.Fprintf(out, "%d. %s - %s (%d)\n", i, name, artist, count)
			i++
		}
		fmt.Fprintln(out)
	}

	return nil
}

func getArtistTags(db *sql.DB, artist string, limit int) (string, error) {
	if limit <= 0 {
		return "", nil
	}
	rows, err := db.Query("SELECT tag FROM ArtistTag WHERE artist = ? ORDER BY count DESC LIMIT ?", artist, limit)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return "", err
		}
		tags = append(tags, tag)
	}
	return strings.Join(tags, ", "), nil
}

func getAlbumTags(db *sql.DB, artist, album string, limit int) (string, error) {
	if limit <= 0 {
		return "", nil
	}
	rows, err := db.Query("SELECT tag FROM AlbumTag WHERE artist = ? AND album = ? ORDER BY count DESC LIMIT ?", artist, album, limit)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return "", err
		}
		tags = append(tags, tag)
	}
	return strings.Join(tags, ", "), nil
}
