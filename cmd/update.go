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
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"

	"github.com/ademuri/last-fm-tools/internal/migration"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shkh/lastfm-go/lastfm"
)

var afterString string
var force bool

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetches data from last.fm",
	Long:  `Stores data in a local SQLite database.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := updateDatabase(viper.GetString("database"), force)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&afterString, "after", "", "Only get listening data after this date, in yyyy-mm-dd format")
	updateCmd.Flags().BoolVarP(&force, "force", "f", false, "Get all listening data, regardless of what's already present (idempotent)")
}

func updateDatabase(dbPath string, force bool) error {
	var after time.Time
	var err error
	if len(afterString) > 0 {
		after, err = time.Parse("2006-01-02", afterString)
		if err != nil {
			return fmt.Errorf("--after: %w", err)
		}
	}

	user := strings.ToLower(viper.GetString("user"))
	database, err := createDatabase(dbPath)
	if err != nil {
		return fmt.Errorf("updateDatabase: %w", err)
	}

	lastfm_client := lastfm.New(lastFmApiKey, lastFmSecret)

	err = createUser(database, user)
	if err != nil {
		return fmt.Errorf("updateDatabase: %w", err)
	}

	latestListen, err := getLatestListen(database, user)
	if err != nil {
		return fmt.Errorf("updateDatabase: %w", err)
	}
	fmt.Printf("Latest local listening data is from %s\n", latestListen.Format("2006-01-02"))

	fmt.Printf("Updating database for %q\n", user)
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)
	page := 1 // First page is 1
	pages := 0
	for {
		var recent_tracks lastfm.UserGetRecentTracks
		err := retry.Do(
			func() error {
				var err error
				recent_tracks, err = lastfm_client.User.GetRecentTracks(lastfm.P{
					"limit": 200,
					"page":  page,
					"user":  user,
				})
				return err
			},
			retry.RetryIf(func(err error) bool {
				if lerr, ok := err.(*lastfm.LastfmError); ok {
					if lerr.Code/100 == 5 {
						fmt.Printf("last.fm errored, retrying: %w", lerr)
						return true
					}
					return false
				} else {
					return false
				}
			}),
		)
		if err != nil {
			return fmt.Errorf("updateDatabase: %w", err)
		}

		if pages == 0 {
			pages = recent_tracks.TotalPages
		}

		err = insertRecentTracks(database, user, recent_tracks)
		if err != nil {
			return fmt.Errorf("updateDatabase: %w", err)
		}
		fmt.Printf("Downloaded page %v of %v\n", page, pages)
		page += 1

		oldestDateUts, err := strconv.ParseInt(recent_tracks.Tracks[len(recent_tracks.Tracks)-1].Date.Uts, 10, 64)
		if err != nil {
			return fmt.Errorf("updateDatabase: %w", err)
		}
		oldestDate := time.Unix(oldestDateUts, 0)
		if !after.IsZero() && oldestDate.Before(after) {
			break
		}
		if page > pages {
			break
		}
		if !force && oldestDate.Before(latestListen.AddDate(0, 0, -7)) {
			fmt.Println("Refreshed back to existing data")
			break
		}

		limiter.Wait(context.Background())
	}

	return nil
}

func createDatabase(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("createDatabase: %w", err)
	}
	err = createTables(database)
	if err != nil {
		return nil, fmt.Errorf("createDatabase: %w", err)
	}

	return database, nil
}

func createTables(db *sql.DB) error {
	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("createTables: %w", err)
	}

	if !exists {
		_, err = db.Exec(migration.Create)
		return err
	}

	return nil
}

func createUser(db *sql.DB, user string) error {
	user_rows, err := db.Query("SELECT name FROM User WHERE name = ?", user)
	if err != nil {
		return fmt.Errorf("createUser(%q): %w", user, err)
	}
	defer user_rows.Close()

	if !user_rows.Next() {
		_, err := db.Exec("INSERT INTO User (name) VALUES (?)", user)
		if err != nil {
			return fmt.Errorf("createUser(%q): %w", user, err)
		}
	}
	return nil
}

func insertRecentTracks(db *sql.DB, user string, recent_tracks lastfm.UserGetRecentTracks) (err error) {
	tx, err := db.Begin()
	if err != nil {
		fmt.Errorf("Creating transaction for insertRecentTracks: %w", err)
	}

	for _, track := range recent_tracks.Tracks {
		err = createArtist(tx, track.Artist.Name)
		if err != nil {
			return fmt.Errorf("insertRecentTracks(page=%v): %w", recent_tracks.Page, err)
		}

		err = createAlbum(tx, track.Artist.Name, track.Album.Name)
		if err != nil {
			return fmt.Errorf("insertRecentTracks(page=%v): %w", recent_tracks.Page, err)
		}

		track_id, err := createTrack(tx, track.Artist.Name, track.Album.Name, track.Name)
		if err != nil {
			return fmt.Errorf("insertRecentTracks(page=%v): %w", recent_tracks.Page, err)
		}
		if track_id == 0 {
			return fmt.Errorf("createTrack(%q, %q, %q) returned 0", track.Artist.Name, track.Album.Name, track.Name)
		}

		err = createListen(tx, user, track_id, track.Date.Uts)
		if err != nil {
			return fmt.Errorf("insertRecentTracks(page=%v): %w", recent_tracks.Page, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Comming transaction in insertRecentTracks: %w", err)
	}

	return nil
}

func createArtist(db *sql.Tx, name string) (err error) {
	artist_exists, err := db.Query("SELECT name FROM Artist WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("createArtist(%q): %w", name, err)
	}
	defer artist_exists.Close()

	if !artist_exists.Next() {
		_, err := db.Exec("INSERT INTO Artist (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("createArtist(%q): %w", name, err)
		}
	}

	return nil
}

func createAlbum(db *sql.Tx, artist string, name string) (err error) {
	album_exists, err := db.Query("SELECT name FROM Album WHERE artist = ? AND name = ?", artist, name)
	if err != nil {
		return fmt.Errorf("createAlbum(%q, %q): %w", artist, name, err)
	}
	defer album_exists.Close()

	if !album_exists.Next() {
		_, err := db.Exec("INSERT INTO Album (artist, name) VALUES (?, ?)", artist, name)
		if err != nil {
			return fmt.Errorf("createAlbum(%q, %q): %w", artist, name, err)
		}
	}

	return nil
}

func createTrack(db *sql.Tx, artist string, album string, name string) (id int64, err error) {
	track_exists, err := db.Query("SELECT id FROM Track WHERE artist = ? AND album = ? AND name = ?", artist, album, name)
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}
	defer track_exists.Close()

	if track_exists.Next() {
		var id int64
		track_exists.Scan(&id)
		return id, nil
	}

	result, err := db.Exec("INSERT INTO Track (artist, album, name) VALUES (?, ?, ?)", artist, album, name)
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}

	track_id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("createTrack(%q, %q, %q): %w", artist, album, name, err)
	}

	return track_id, nil
}

func createListen(db *sql.Tx, user string, track_id int64, datetime string) (err error) {
	listen_exists, err := db.Query("SELECT id FROM Listen WHERE user = ? AND date = ? AND track = ?", user, datetime, track_id)
	if err != nil {
		return fmt.Errorf("createListen(%q, %q, %q): %w", user, track_id, datetime, err)
	}
	defer listen_exists.Close()

	if listen_exists.Next() {
		return nil
	}

	_, err = db.Exec("INSERT INTO Listen (user, track, date) VALUES (?, ?, ?)", user, track_id, datetime)
	if err != nil {
		return fmt.Errorf("createListen(%q, %q, %q): %w", user, track_id, datetime, err)
	}
	return nil
}

func getLatestListen(db *sql.DB, user string) (date time.Time, err error) {
	query, err := db.Query("SELECT date FROM Listen WHERE user = ? ORDER BY Date desc LIMIT 1", user)
	if err != nil {
		err = fmt.Errorf("getLatestListen(%q): %w", user, err)
		return
	}
	defer query.Close()

	if !query.Next() {
		return
	}

	query.Scan(&date)
	return
}
