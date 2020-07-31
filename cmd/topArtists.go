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
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var topArtistsNumber int
var topArtistsCmd = &cobra.Command{
	Use:   "top-artists [from] [to (optional)]",
	Short: "Gets the user's top artists",
	Long:  `Uses the specified date or date range. Date strings look like 'yyyy', 'yyyy-mm', or 'yyyy-mm-dd'.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := printTopArtists(viper.GetString("database"), topArtistsNumber, args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(topArtistsCmd)

	topArtistsCmd.Flags().IntVarP(&topArtistsNumber, "number", "n", 10, "number of results to return")
}

func printTopArtists(dbPath string, numToReturn int, args []string) error {
	start, end, err := parseDateRangeFromArgs(args)

	config := AnalyserConfig{numToReturn, 0}
	out, err := TopArtistsAnalyzer{}.SetConfig(config).GetResults(dbPath, viper.GetString("user"), start, end)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

type TopArtistsAnalyzer struct {
	Config AnalyserConfig
}

func (t TopArtistsAnalyzer) SetConfig(config AnalyserConfig) TopArtistsAnalyzer {
	t.Config = config
	return t
}

func (t TopArtistsAnalyzer) GetName() string {
	return "Top artists"
}

func (t TopArtistsAnalyzer) GetResults(dbPath string, user string, start time.Time, end time.Time) (analysis Analysis, err error) {
	db, err := openDb(dbPath)
	if err != nil {
		err = fmt.Errorf("printTopArtists: %w", err)
		return
	}

	exists, err := dbExists(db)
	if err != nil {
		err = fmt.Errorf("printTopArtists: %w", err)
		return
	}
	if !exists {
		err = fmt.Errorf("Database doesn't exist - run update first.")
		return
	}

	const countQueryString = `
	SELECT Track.artist, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist
	ORDER BY COUNT(*) DESC
	;
	`
	countQuery, err := db.Query(countQueryString, user, start.Unix(), end.Unix())
	if err != nil {
		err = fmt.Errorf("printTopArtists: %w", err)
		return
	}

	numArtists := 0
	var numListens int64 = 0
	analysis.results = [][]string{{"Artist", "Listens"}}
	for countQuery.Next() {
		artist := make([]string, 2)
		countQuery.Scan(&artist[0], &artist[1])
		numArtists += 1
		var listens int64
		listens, err = strconv.ParseInt(artist[1], 10, 64)
		if err != nil {
			err = fmt.Errorf("counting listens: %w", err)
			return
		}

		if (t.Config.NumToReturn == 0 || numArtists <= t.Config.NumToReturn) && (t.Config.FilterThreshold == 0 || listens > t.Config.FilterThreshold) {
			analysis.results = append(analysis.results, artist)
		}

		numListens += listens
	}

	const dateFormat = "2006-01-02"
	analysis.summary = fmt.Sprintf("Found %d artists and %d listens from %s to %s\n",
		numArtists, numListens, start.Format(dateFormat), end.Format(dateFormat))

	return
}
