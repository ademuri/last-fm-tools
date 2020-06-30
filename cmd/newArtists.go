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
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newArtistsNumber int
var newArtistsCmd = &cobra.Command{
	Use:   "new-artists [from] [to (optional)]",
	Short: "Gets new artists for the given time period",
	Long:  `Uses the specified date or date range. Date strings look like 'yyyy', 'yyyy-mm', or 'yyyy-mm-dd'.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := printNewArtists(viper.GetString("database"), newArtistsNumber, args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

type ArtistCount struct {
	Artist string
	Count  int64
}

func init() {
	rootCmd.AddCommand(newArtistsCmd)

	newArtistsCmd.Flags().IntVarP(&newArtistsNumber, "number", "n", 0, "number of results to return")
}

func printNewArtists(dbPath string, numToReturn int, args []string) error {
	start, end, err := parseDateRangeFromArgs(args)

	out, err := NewArtistsAnalyzer{}.GetResults(dbPath, numToReturn, start, end)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

type NewArtistsAnalyzer struct {
}

func (t NewArtistsAnalyzer) GetName() string {
	return "New artists"
}

func (t NewArtistsAnalyzer) GetResults(dbPath string, numToReturn int, start time.Time, end time.Time) (string, error) {
	db, err := openDb(dbPath)
	if err != nil {
		return "", fmt.Errorf("printNewArtists: %w", err)
	}

	exists, err := dbExists(db)
	if err != nil {
		return "", fmt.Errorf("printNewArtists: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("Database doesn't exist - run update first.")
	}

	out := new(bytes.Buffer)
	user := strings.ToLower(viper.GetString("user"))
	var zeroTime time.Time
	prevArtists, err := getArtistsForPeriod(db, user, zeroTime, start)
	if err != nil {
		return "", fmt.Errorf("printNewArtists: %w", err)
	}
	curArtists, err := getArtistsForPeriod(db, user, start, end)
	if err != nil {
		return "", fmt.Errorf("printNewArtists: %w", err)
	}
	fmt.Fprintf(out, "Got %d prev artists, %d cur artists\n", len(prevArtists), len(curArtists))

	counts := make([]ArtistCount, 0)
	for artist, count := range curArtists {
		if prevListens, ok := prevArtists[artist]; (!ok || prevListens < 5) && curArtists[artist] > 5 {
			counts = append(counts, ArtistCount{artist, count})
		}
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Artist"})
	n := 0
	for _, count := range counts {
		table.Append([]string{count.Artist, strconv.FormatInt(count.Count, 10)})
		n += 1
		if numToReturn > 0 && n > numToReturn {
			break
		}
	}
	table.Render()

	return out.String(), nil
}

func getArtistsForPeriod(db *sql.DB, user string, start time.Time, end time.Time) (map[string]int64, error) {
	const countQueryString = `
	SELECT Track.artist, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist
	;
	`
	artists := make(map[string]int64)

	countQuery, err := db.Query(countQueryString, user, start.Unix(), end.Unix())
	if err != nil {
		return artists, fmt.Errorf("getArtistsForPeriod: %w", err)
	}
	defer countQuery.Close()

	for countQuery.Next() {
		var artist string
		var count int64
		err = countQuery.Scan(&artist, &count)
		if err != nil {
			return artists, fmt.Errorf("getArtistsForPeriod: %w", err)
		}
		artists[artist] = count
	}

	return artists, nil
}
