package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shkh/lastfm-go/lastfm"
	"io/ioutil"
	"secrets"
)

func main() {
	database, err := sql.Open("sqlite3", "./my.db")
	if err != nil {
		fmt.Println("Error opening sqlite3 database", err)
		return
	}
	err = createTables(database)
	if err != nil {
		fmt.Println("Error creating database", err)
	}

	lastfm_client := lastfm.New(secrets.LastFmApiKey, secrets.LastFmSecret)

	recent_tracks, err := lastfm_client.User.GetRecentTracks(lastfm.P{
		"user": secrets.LastFmUser,
	})
	if err != nil {
		fmt.Println("Error getting recent tracks: %s", err)
		return
	}

	fmt.Println("Got", recent_tracks.Total, "tracks")
}

func createTables(db *sql.DB) (err error) {
	data, err := ioutil.ReadFile("sql/create-tables.sql")
	if err != nil {
		return err
	}

	statement, err := db.Prepare(string(data))
	if err != nil {
		return err
	}
	statement.Exec()

	return nil
}
