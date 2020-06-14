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
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newAlbumsNumber int
var newAlbumsCmd = &cobra.Command{
	Use:   "new-albums [from] [to (optional)]",
	Short: "Gets new albums for the given time period",
	Long:  `Uses the specified date or date range. Date strings look like 'yyyy', 'yyyy-mm', or 'yyyy-mm-dd'.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := printNewAlbums(viper.GetString("database"), args, newAlbumsNumber)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

type ArtistAlbum struct {
	Artist string
	Album  string
}

type AlbumCount struct {
	Album ArtistAlbum
	Count int64
}

func init() {
	rootCmd.AddCommand(newAlbumsCmd)

	newAlbumsCmd.Flags().IntVarP(&newAlbumsNumber, "number", "n", 0, "number of results to return")
}

func printNewAlbums(dbPath string, args []string, numToReturn int) error {
	start, end, err := parseDateRangeFromArgs(args)

	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("printNewAlbums: %w", err)
	}

	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("printNewAlbums: %w", err)
	}
	if !exists {
		return fmt.Errorf("Database doesn't exist - run update first.")
	}

	user := strings.ToLower(viper.GetString("user"))
	var zeroTime time.Time
	prevAlbums, err := getAlbumsForPeriod(db, user, zeroTime, start)
	if err != nil {
		return fmt.Errorf("printNewAlbums: %w", err)
	}
	curAlbums, err := getAlbumsForPeriod(db, user, start, end)
	if err != nil {
		return fmt.Errorf("printNewAlbums: %w", err)
	}
	fmt.Printf("Got %d prev albums, %d cur albums\n", len(prevAlbums), len(curAlbums))

	counts := make([]AlbumCount, 0)
	for album, count := range curAlbums {
		if prevListens, ok := prevAlbums[album]; (!ok || prevListens < 5) && curAlbums[album] > 5 {
			counts = append(counts, AlbumCount{album, count})
		}
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Artist", "Album", "Listens"})
	n := 0
	for _, count := range counts {
		table.Append([]string{count.Album.Artist, count.Album.Album, strconv.FormatInt(count.Count, 10)})
		n += 1
		if numToReturn > 0 && n > numToReturn {
			break
		}
	}
	table.Render()

	return nil
}

func getAlbumsForPeriod(db *sql.DB, user string, start time.Time, end time.Time) (map[ArtistAlbum]int64, error) {
	const countQueryString = `
	SELECT Track.artist, Track.album, COUNT(Listen.id)
	FROM Listen
	INNER JOIN Track ON Track.id = Listen.track
	WHERE user = ?
	AND Listen.date BETWEEN ? AND ?
	GROUP BY Track.artist, Track.album
	;
	`
	albums := make(map[ArtistAlbum]int64)

	countQuery, err := db.Query(countQueryString, user, start.Unix(), end.Unix())
	if err != nil {
		return albums, fmt.Errorf("getAlbumsForPeriod: %w", err)
	}
	defer countQuery.Close()

	for countQuery.Next() {
		var album ArtistAlbum
		var count int64
		err = countQuery.Scan(&album.Artist, &album.Album, &count)
		if err != nil {
			return albums, fmt.Errorf("getAlbumsForPeriod: %w", err)
		}
		albums[album] = count
	}

	return albums, nil
}
