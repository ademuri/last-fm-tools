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
	"strings"

	"github.com/spf13/cobra"

	"github.com/ademuri/last-fm-tools/internal/migration"
	"github.com/ademuri/last-fm-tools/internal/secrets"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shkh/lastfm-go/lastfm"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetches data from last.fm",
	Long:  `Stores data in a local SQLite database.`,
	Run: func(cmd *cobra.Command, args []string) {
		updateDatabase()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func updateDatabase() {
	user := strings.ToLower(lastFmUser)
	database, err := createDatabase()
	if err != nil {
		fmt.Println("Error creating database", err)
		return
	}

	lastfm_client := lastfm.New(secrets.LastFmApiKey, secrets.LastFmSecret)

	err = createUser(database, user)
	if err != nil {
		fmt.Println("Error creating user", err)
		return
	}

	recent_tracks, err := lastfm_client.User.GetRecentTracks(lastfm.P{
		"user": user,
	})
	if err != nil {
		fmt.Println("Error getting recent tracks: ", err)
		return
	}

	fmt.Println("Got", recent_tracks.Total, "tracks")
	err = insertRecentTracks(database, recent_tracks)
	if err != nil {
		fmt.Println("Error inserting tracks:", err)
	}
}

func createDatabase() (*sql.DB, error) {
	database, err := sql.Open("sqlite3", "./my.db")
	if err != nil {
		fmt.Println("Error opening sqlite3 database", err)
		return nil, err
	}
	err = createTables(database)
	if err != nil {
		fmt.Println("Error creating database", err)
		return nil, err
	}

	return database, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(migration.Create)
	return err
}

func createUser(db *sql.DB, user string) (err error) {
	user_rows, err := db.Query("SELECT id FROM User WHERE name = ?", user)
	defer user_rows.Close()
	if err != nil {
		fmt.Println("Error finding existing user")
		return err
	}

	if !user_rows.Next() {
		_, err := db.Exec("INSERT INTO User (name) VALUES (?)", user)
		if err != nil {
			fmt.Println("Error creating user")
			return err
		}
	}
	return nil
}

func insertRecentTracks(db *sql.DB, recent_tracks lastfm.UserGetRecentTracks) (err error) {
	for _, track := range recent_tracks.Tracks {
		err := createArtist(db, track.Artist.Name)
		if err != nil {
			fmt.Println("Error creating artist:", track.Artist.Name)
			return err
		}

		err = createAlbum(db, track.Artist.Name, track.Album.Name)
		if err != nil {
			fmt.Println("Error creating album:", track.Album.Name)
			return err
		}

		track_id, err := createTrack(db, track.Artist.Name, track.Album.Name, track.Name)
		if err != nil {
			fmt.Println("Error creating track:", track.Name)
			return err
		}

		err = createListen(db, track_id, track.Date.Uts)
		if err != nil {
			fmt.Println("Error creating listen:", track.Date.Uts)
			return err
		}
	}

	return nil
}

func createArtist(db *sql.DB, name string) (err error) {
	artist_exists, err := db.Query("SELECT name FROM Artist WHERE name = ?", name)
	defer artist_exists.Close()
	if err != nil {
		return err
	}

	if !artist_exists.Next() {
		_, err := db.Exec("INSERT INTO Artist (name) VALUES (?)", name)
		if err != nil {
			return err
		}
	}

	return nil
}

func createAlbum(db *sql.DB, artist string, name string) (err error) {
	album_exists, err := db.Query("SELECT name FROM Album WHERE name = ? AND artist = ?", name, artist)
	defer album_exists.Close()
	if err != nil {
		return err
	}

	if !album_exists.Next() {
		_, err := db.Exec("INSERT INTO Album (name, artist) VALUES (?, ?)", name, artist)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTrack(db *sql.DB, artist string, album string, name string) (id int64, err error) {
	track_exists, err := db.Query("SELECT id FROM Track WHERE name = ? AND artist = ? AND album = ?", name, artist, album)
	defer track_exists.Close()
	if err != nil {
		return 0, err
	}

	if track_exists.Next() {
		var id int64
		track_exists.Scan(id)
		return id, nil
	}

	result, err := db.Exec("INSERT INTO Track (name, artist, album) VALUES (?, ?, ?)", name, artist, album)
	if err != nil {
		return 0, err
	}

	track_id, _ := result.LastInsertId()
	return track_id, nil
}

func createListen(db *sql.DB, track_id int64, datetime string) (err error) {
	listen_exists, err := db.Query("SELECT id FROM Listen WHERE track = ? AND date = ?", track_id, datetime)
	defer listen_exists.Close()
	if err != nil {
		return err
	}

	if listen_exists.Next() {
		return nil
	}

	_, err = db.Exec("INSERT INTO Listen (track, date) VALUES (?, ?)", track_id, datetime)
	return err
}
