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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"

	"github.com/ademuri/last-fm-tools/internal/store"
	"github.com/ademuri/lastfm-go/lastfm"
)

type UpdateConfig struct {
	DbPath            string
	User              string
	After             string
	Force             bool
	TagUpdateInterval time.Duration
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetches data from last.fm",
	Long:  `Stores data in a local SQLite database.`,
	Run: func(cmd *cobra.Command, args []string) {
		intervalStr := viper.GetString("tag-update-interval")
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			fmt.Printf("Invalid tag-update-interval: %v. Using default 1 year.\n", err)
			interval = 24 * 365 * time.Hour
		}

		config := UpdateConfig{
			DbPath:            viper.GetString("database"),
			User:              viper.GetString("user"),
			After:             viper.GetString("after"),
			Force:             viper.GetBool("force"),
			TagUpdateInterval: interval,
		}

		err = updateDatabase(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	var afterString string
	updateCmd.Flags().StringVar(&afterString, "after", "", "Only get listening data after this date, in yyyy-mm-dd format")
	viper.BindPFlag("after", updateCmd.Flags().Lookup("after"))

	var force bool
	updateCmd.Flags().BoolVarP(&force, "force", "f", false, "Get all listening data, regardless of what's already present (idempotent)")
	viper.BindPFlag("force", updateCmd.Flags().Lookup("force"))

	var tagUpdateInterval string
	updateCmd.Flags().StringVar(&tagUpdateInterval, "tag-update-interval", "8760h", "Time duration after which to re-fetch tags (e.g., 24h)")
	viper.BindPFlag("tag-update-interval", updateCmd.Flags().Lookup("tag-update-interval"))
}

func updateDatabase(config UpdateConfig) error {
	var after time.Time
	var err error
	if len(config.After) > 0 {
		after, err = time.Parse("2006-01-02", config.After)
		if err != nil {
			return fmt.Errorf("--after: %w", err)
		}
	}

	user := strings.ToLower(config.User)
	db, err := store.New(config.DbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	lastfmClient := lastfm.New(lastFmApiKey, lastFmSecret)
	lastfmClient.SetUserAgent("last-fm-tools/1.0")

	err = db.CreateUser(user)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	lastUpdated, err := db.GetLastUpdated(user)
	if err != nil {
		return err
	}
	now := time.Now()
	if !lastUpdated.IsZero() && now.Sub(lastUpdated).Hours() < 24 && !config.Force {
		fmt.Printf("User data was already updated in the past 24 hours\n")
		return nil
	}
	fmt.Printf("User data was last updated: %s\n", lastUpdated.Format("2006-01-02"))

	// Session Key
	sessionKey, err := db.GetSessionKey(user)
	if err != nil {
		return err
	}
	if sessionKey != "" {
		lastfmClient.SetSession(sessionKey)
		fmt.Printf("Using session key for user %q\n", user)
	}

	latestListen, err := db.GetLatestListen(user)
	if err != nil {
		return fmt.Errorf("getting latest listen: %w", err)
	}
	fmt.Printf("Latest local listening data is from: %s\n", latestListen.Format("2006-01-02"))

	fmt.Printf("Updating database for %q\n", user)
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)
	page := 1 // First page is 1
	pages := 0
	for {
		var recentTracks lastfm.UserGetRecentTracks
		err := retry.Do(
			func() error {
				var err error
				recentTracks, err = lastfmClient.User.GetRecentTracks(lastfm.P{
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
				}
				return false
			}),
		)
		if err != nil {
			return fmt.Errorf("fetching recent tracks: %w", err)
		}

		if pages == 0 {
			pages = recentTracks.TotalPages
		}

		// Convert to store.TrackImport
		var tracksToImport []store.TrackImport
		for _, t := range recentTracks.Tracks {
			tracksToImport = append(tracksToImport, store.TrackImport{
				Artist:    t.Artist.Name,
				Album:     t.Album.Name,
				TrackName: t.Name,
				DateUTS:   t.Date.Uts,
			})
		}

		err = db.AddRecentTracks(user, tracksToImport)
		if err != nil {
			return fmt.Errorf("inserting recent tracks (page %d): %w", page, err)
		}

		oldestDateUts, err := strconv.ParseInt(recentTracks.Tracks[len(recentTracks.Tracks)-1].Date.Uts, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing date: %w", err)
		}
		oldestDate := time.Unix(oldestDateUts, 0)

		fmt.Printf("Downloaded page %v of %v (oldest: %s)\n", page, pages, oldestDate.Format("2006-01-02"))
		page += 1

		if !after.IsZero() && oldestDate.Before(after) {
			break
		}
		if page > pages {
			break
		}
		if !config.Force && !latestListen.IsZero() && oldestDate.Before(latestListen.AddDate(0, 0, -7)) {
			fmt.Println("Refreshed back to existing data")
			break
		}

		limiter.Wait(context.Background())
	}

	fmt.Println("Updating tags...")
	err = updateTags(db, lastfmClient, config.TagUpdateInterval)
	if err != nil {
		return err
	}

	err = db.SetLastUpdated(user, now)
	if err != nil {
		return err
	}

	return nil
}

func updateTags(db *store.Store, lastfmClient *lastfm.Api, interval time.Duration) error {
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

	err := updateArtistTags(db, lastfmClient, limiter, interval)
	if err != nil {
		return fmt.Errorf("updateArtistTags: %w", err)
	}

	err = updateAlbumTags(db, lastfmClient, limiter, interval)
	if err != nil {
		return fmt.Errorf("updateAlbumTags: %w", err)
	}

	return nil
}

func updateArtistTags(db *store.Store, client *lastfm.Api, limiter *rate.Limiter, interval time.Duration) error {
	artists, err := db.GetArtistsNeedingTagUpdate(interval)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d artists needing tag updates\n", len(artists))

	for i, artist := range artists {
		fmt.Printf("[%d/%d] Fetching tags for artist: %s\n", i+1, len(artists), artist)
		limiter.Wait(context.Background())

		var topTags lastfm.ArtistGetTopTags
		err := retry.Do(
			func() error {
				var err error
				topTags, err = client.Artist.GetTopTags(lastfm.P{
					"artist":      artist,
					"autocorrect": 1,
				})
				return err
			},
			retry.RetryIf(func(err error) bool {
				if lerr, ok := err.(*lastfm.LastfmError); ok {
					if lerr.Code/100 == 5 {
						fmt.Printf("last.fm errored, retrying: %w\n", lerr)
						return true
					}
				}
				return false
			}),
		)
		if err != nil {
			fmt.Printf("Error fetching tags for artist %s: %v\n", artist, err)
			continue
		}

		var tags []string
		var counts []int
		for _, t := range topTags.Tags {
			tags = append(tags, t.Name)
			c, _ := strconv.Atoi(t.Count)
			counts = append(counts, c)
		}

		if err := db.SaveArtistTags(artist, tags, counts); err != nil {
			return fmt.Errorf("saving tags for artist %s: %w", artist, err)
		}
	}

	return nil
}

func updateAlbumTags(db *store.Store, client *lastfm.Api, limiter *rate.Limiter, interval time.Duration) error {
	albums, err := db.GetAlbumsNeedingTagUpdate(interval)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d albums needing tag updates\n", len(albums))

	for i, alb := range albums {
		fmt.Printf("[%d/%d] Fetching tags for album: %s - %s\n", i+1, len(albums), alb.Artist, alb.Name)
		limiter.Wait(context.Background())

		var topTags lastfm.AlbumGetTopTags
		err := retry.Do(
			func() error {
				var err error
				topTags, err = client.Album.GetTopTags(lastfm.P{
					"artist":      alb.Artist,
					"album":       alb.Name,
					"autocorrect": 1,
				})
				return err
			},
			retry.RetryIf(func(err error) bool {
				if lerr, ok := err.(*lastfm.LastfmError); ok {
					if lerr.Code/100 == 5 {
						fmt.Printf("last.fm errored, retrying: %w\n", lerr)
						return true
					}
				}
				return false
			}),
		)
		if err != nil {
			fmt.Printf("Error fetching tags for album %s - %s: %v\n", alb.Artist, alb.Name, err)
			continue
		}

		var tags []string
		var counts []int
		for _, t := range topTags.Tags {
			tags = append(tags, t.Name)
			c, _ := strconv.Atoi(t.Count)
			counts = append(counts, c)
		}

		if err := db.SaveAlbumTags(alb.Artist, alb.Name, tags, counts); err != nil {
			return fmt.Errorf("saving tags for album %s - %s: %w", alb.Artist, alb.Name, err)
		}
	}

	return nil
}