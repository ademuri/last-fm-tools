/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var topAlbumsCmd = &cobra.Command{
	Use:   "top-albums [date string]",
	Short: "Gets the user's top albums",
	Long:  `TODO`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := getTopAlbums(viper.GetString("database"), args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(topAlbumsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// topAlbumsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// topAlbumsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type AlbumCount struct {
	Artist string
	Name   string
	Count  int64
}

func getTopAlbums(dbPath string, ds string) error {
	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("getTopAlbums: %w", err)
	}

	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("getTopAlbums: %w", err)
	}
	if !exists {
		return fmt.Errorf("Database doesn't exist - run update first.")
	}

	start, err := time.Parse("2006-01", ds)
	if err != nil {
		return fmt.Errorf("getTopAlbums(%q): %w", ds, err)
	}
	end := start.AddDate(0, 1, 0)

	user := strings.ToLower(viper.GetString("user"))

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
		return fmt.Errorf("getTopAlbums(%q): %w", ds, err)
	}

	n := 0
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Artist", "Album", "Listens"})
	for countQuery.Next() {
		album := make([]string, 3)
		countQuery.Scan(&album[0], &album[1], &album[2])
		if n < 10 {
			table.Append(album)
		}
		n += 1
	}
	table.Render()
	fmt.Printf("Found %d albums\n", n)

	return nil
}
