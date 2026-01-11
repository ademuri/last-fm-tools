package analysis

// Report is the top-level structure for the music taste report.
type Report struct {
	Metadata          ProfileMetadata   `yaml:"profile_metadata"`
	CurrentTaste      TasteProfile      `yaml:"current_taste"`
	HistoricalBaseline TasteProfile      `yaml:"historical_baseline"`
	TasteDrift        TasteDrift        `yaml:"taste_drift"`
	ListeningPatterns ListeningPatterns `yaml:"listening_patterns"`
}

type ProfileMetadata struct {
	GeneratedDate    string `yaml:"generated_date"`
	TotalScrobbles   int64  `yaml:"total_scrobbles"`
	TotalArtists     int    `yaml:"total_artists"`
	ListeningStyle   string `yaml:"listening_style"`
	CurrentPeriod    string `yaml:"current_period"`
	HistoricalPeriod string `yaml:"historical_period"`
}

type TasteProfile struct {
	TopArtists []ArtistStat `yaml:"top_artists"`
	TopAlbums  []AlbumStat  `yaml:"top_albums,omitempty"`
	TopTags    []TagStat    `yaml:"top_tags"`
}

type ArtistStat struct {
	Name               string   `yaml:"name"`
	Scrobbles          int64    `yaml:"scrobbles"`
	InHistoricalBaseline bool   `yaml:"in_historical_baseline,omitempty"`
	InCurrentTaste     bool     `yaml:"in_current_taste,omitempty"`
	PeakYears          string   `yaml:"peak_years,omitempty"`
	TopAlbums          []string `yaml:"top_albums,omitempty"`
	PrimaryTags        []string `yaml:"primary_tags"`
}

type AlbumStat struct {
	Title     string   `yaml:"title"`
	Artist    string   `yaml:"artist"`
	Scrobbles int64    `yaml:"scrobbles"`
	Tags      []string `yaml:"tags"`
}

type TagStat struct {
	Tag    string  `yaml:"tag"`
	Weight float64 `yaml:"weight"`
}

type TasteDrift struct {
	DeclinedTags []DriftTag `yaml:"declined_tags"`
	EmergedTags  []DriftTag `yaml:"emerged_tags"`
}

type DriftTag struct {
	Tag              string  `yaml:"tag"`
	HistoricalWeight float64 `yaml:"historical_weight"`
	CurrentWeight    float64 `yaml:"current_weight"`
}

type ListeningPatterns struct {
	AlbumsPerArtistMedian   float64 `yaml:"albums_per_artist_median"`
	AlbumsPerArtistAverage  float64 `yaml:"albums_per_artist_average"`
	NewArtistsInLast12Month int     `yaml:"new_artists_in_last_12_month"`
	RepeatListeningRatio    float64 `yaml:"repeat_listening_ratio"`
}