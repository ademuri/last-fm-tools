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
	"time"

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

	config := AnalyserConfig{numToReturn, 0}
	analyzer := &NewArtistsAnalyzer{}
	out, err := analyzer.SetConfig(config).GetResults(dbPath, viper.GetString("user"), start, end)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

type NewArtistsAnalyzer struct {
	Config AnalyserConfig
}

func (t *NewArtistsAnalyzer) SetConfig(config AnalyserConfig) *NewArtistsAnalyzer {
	t.Config = config
	return t
}

func (t *NewArtistsAnalyzer) Configure(params map[string]string) error {
	if val, ok := params["n"]; ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid value for 'n': %v", err)
		}
		t.Config.NumToReturn = n
	}
	if val, ok := params["min"]; ok {
		min, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for 'min': %v", err)
		}
		t.Config.FilterThreshold = min
	}
	return nil
}

func (t *NewArtistsAnalyzer) GetName() string {
	return "New artists"
}

func (t *NewArtistsAnalyzer) GetResults(dbPath string, user string, start time.Time, end time.Time) (analysis Analysis, err error) {
	db, err := openDb(dbPath)
	if err != nil {
		err = fmt.Errorf("printNewArtists: %w", err)
		return
	}

	exists, err := dbExists(db)
	if err != nil {
		err = fmt.Errorf("printNewArtists: %w", err)
		return
	}
	if !exists {
		err = fmt.Errorf("Database doesn't exist - run update first.")
		return
	}

	out := new(bytes.Buffer)
	var zeroTime time.Time
	prevArtists, err := getArtistsForPeriod(db, user, zeroTime, start)
	if err != nil {
		err = fmt.Errorf("printNewArtists: %w", err)
		return
	}
	curArtists, err := getArtistsForPeriod(db, user, start, end)
	if err != nil {
		err = fmt.Errorf("printNewArtists: %w", err)
		return
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

	analysis.results = append(analysis.results, []string{"Artist", "Listens"})
	n := 0
	var numListens int64 = 0
	for _, count := range counts {
		if (t.Config.NumToReturn == 0 || n <= t.Config.NumToReturn) && (t.Config.FilterThreshold == 0 || count.Count > t.Config.FilterThreshold) {
			analysis.results = append(analysis.results, []string{count.Artist, strconv.FormatInt(count.Count, 10)})
		}
		n += 1
		numListens += count.Count
	}
	const dateFormat = "2006-01-02"
	analysis.summary = fmt.Sprintf("Found %d new artists with %d listens from %s to %s\n",
		n, numListens, start.Format(dateFormat), end.Format(dateFormat))

	return
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
