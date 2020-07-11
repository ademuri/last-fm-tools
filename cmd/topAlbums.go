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
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var topAlbumsNumber int
var topAlbumsCmd = &cobra.Command{
	Use:   "top-albums [from] [to (optional)]",
	Short: "Gets the user's top albums",
	Long:  `Uses the specified date or date range. Date strings look like 'yyyy', 'yyyy-mm', or 'yyyy-mm-dd'.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := printTopAlbums(viper.GetString("database"), topAlbumsNumber, args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(topAlbumsCmd)

	topAlbumsCmd.Flags().IntVarP(&topAlbumsNumber, "number", "n", 10, "number of results to return")
}

func printTopAlbums(dbPath string, numToReturn int, args []string) error {
	start, end, err := parseDateRangeFromArgs(args)

	out, err := TopAlbumsAnalyzer{}.GetResults(dbPath, viper.GetString("user"), numToReturn, start, end)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

type TopAlbumsAnalyzer struct {
}

func (t TopAlbumsAnalyzer) GetName() string {
	return "Top albums"
}

func (t TopAlbumsAnalyzer) GetResults(dbPath string, user string, numToReturn int, start time.Time, end time.Time) (string, error) {
	db, err := openDb(dbPath)
	if err != nil {
		return "", fmt.Errorf("getTopAlbums: %w", err)
	}

	exists, err := dbExists(db)
	if err != nil {
		return "", fmt.Errorf("getTopAlbums: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("Database doesn't exist - run update first.")
	}

	const countQueryString = `
	SELECT Track.artist, Track.album, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist, Track.album
	ORDER BY COUNT(*) DESC
	;
	`
	countQuery, err := db.Query(countQueryString, user, start.Unix(), end.Unix())
	if err != nil {
		return "", fmt.Errorf("getTopAlbums: %w", err)
	}

	numAlbums := 0
	var numListens int64 = 0
	out := new(bytes.Buffer)
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Artist", "Album", "Listens"})
	for countQuery.Next() {
		album := make([]string, 3)
		countQuery.Scan(&album[0], &album[1], &album[2])
		if numAlbums < numToReturn {
			table.Append(album)
		}
		numAlbums += 1
		listens, err := strconv.ParseInt(album[2], 10, 64)
		if err != nil {
			return "", fmt.Errorf("counting listens: %w", err)
		}
		numListens += listens
	}
	table.Render()
	const dateFormat = "2006-01-02"
	fmt.Fprintf(out, "Found %d albums and %d listens from %s to %s\n",
		numAlbums, numListens, start.Format(dateFormat), end.Format(dateFormat))

	return out.String(), nil
}
