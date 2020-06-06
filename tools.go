package main

import "fmt"
import "github.com/shkh/lastfm-go/lastfm"
import "secrets"

func main() {
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
